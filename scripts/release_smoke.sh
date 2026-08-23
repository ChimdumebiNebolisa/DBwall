#!/usr/bin/env bash
# Release-archive smoke check. Mirrors the portable build steps of
# .github/workflows/release.yml, verifies archive structure, checksums, and
# that the produced binary behaves (version + a review decision).
#
# Usage:
#   scripts/release_smoke.sh
# Optional:
#   DBWALL_SMOKE_FULL_BIN=/path/to/native-full-mode-binary  additionally packages
#     and verifies a full-mode archive the way the release workflow does for
#     linux_amd64_full (useful on hosts where CGO cross-compiling to Linux is
#     unavailable; the binary itself is only verified when it can run locally).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${DBWALL_SMOKE_VERSION:-v0.0.0-smoke}"
LDFLAGS="-X github.com/ChimdumebiNebolisa/DBwall/internal/version.Version=${VERSION}"
DIST="$(mktemp -d)"
trap 'rm -rf "$DIST"' EXIT

cd "${ROOT}"

builds=(
  "linux amd64 tar.gz"
  "darwin amd64 tar.gz"
  "darwin arm64 tar.gz"
  "windows amd64 zip"
)

create_zip() {
  # Mirrors release.yml's `zip -rq`; falls back to bsdtar for hosts without zip.
  local src_dir="$1" out_zip="$2"
  if command -v zip >/dev/null 2>&1; then
    (cd "${src_dir}" && zip -rq "${out_zip}" .)
  elif command -v bsdtar >/dev/null 2>&1; then
    bsdtar -a -cf "${out_zip}" -C "${src_dir}" .
  elif [[ -x /c/Windows/System32/tar.exe ]] && /c/Windows/System32/tar.exe --version | grep -q bsdtar; then
    /c/Windows/System32/tar.exe -a -cf "$(cygpath -w "${out_zip}")" -C "$(cygpath -w "${src_dir}")" .
  else
    echo "no zip-capable tool found" >&2
    exit 1
  fi
}

for build in "${builds[@]}"; do
  read -r GOOS GOARCH ARCHIVE <<<"${build}"
  BIN="dbguard"
  if [[ "${GOOS}" == "windows" ]]; then
    BIN="dbguard.exe"
  fi
  OUTDIR="${DIST}/dbguard_${VERSION}_${GOOS}_${GOARCH}"
  mkdir -p "${OUTDIR}"
  CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -ldflags "${LDFLAGS}" -o "${OUTDIR}/${BIN}" ./cmd/dbguard
  cp README.md LICENSE "${OUTDIR}/"
  if [[ "${ARCHIVE}" == "zip" ]]; then
    create_zip "${OUTDIR}" "${DIST}/dbguard_${VERSION}_${GOOS}_${GOARCH}.zip"
  else
    tar -C "${OUTDIR}" -czf "${DIST}/dbguard_${VERSION}_${GOOS}_${GOARCH}.tar.gz" .
  fi
  rm -rf "${OUTDIR}"
done

(cd "${DIST}" && sha256sum -- *.tar.gz *.zip > checksums.txt)

echo "== checksum roundtrip"
(cd "${DIST}" && sha256sum -c checksums.txt) >/dev/null

echo "== archive structure"
normalize_tar_list() { tar -tzf "$1" | sed 's|^\./||'; }
for f in linux_amd64 darwin_amd64 darwin_arm64; do
  entries="$(normalize_tar_list "${DIST}/dbguard_${VERSION}_${f}.tar.gz")"
  echo "${entries}" | grep -qx 'dbguard' || { echo "missing dbguard in ${f}" >&2; exit 1; }
  echo "${entries}" | grep -qx 'README.md' || { echo "missing README.md in ${f}" >&2; exit 1; }
  echo "${entries}" | grep -qx 'LICENSE' || { echo "missing LICENSE in ${f}" >&2; exit 1; }
done

extract_zip() {
  local zip_path="$1" dest="$2"
  mkdir -p "${dest}"
  if command -v unzip >/dev/null 2>&1; then
    unzip -q "${zip_path}" -d "${dest}"
  elif command -v bsdtar >/dev/null 2>&1; then
    bsdtar -xf "${zip_path}" -C "${dest}"
  elif [[ -x /c/Windows/System32/tar.exe ]] && /c/Windows/System32/tar.exe --version | grep -q bsdtar; then
    /c/Windows/System32/tar.exe -xf "$(cygpath -w "${zip_path}")" -C "$(cygpath -w "${dest}")"
  else
    echo "no zip-capable extraction tool found" >&2
    exit 1
  fi
}

ZIP_PATH="${DIST}/dbguard_${VERSION}_windows_amd64.zip"
extract_zip "${ZIP_PATH}" "${DIST}/winzip"
[[ -f "${DIST}/winzip/dbguard.exe" ]] || { echo "zip missing dbguard.exe" >&2; exit 1; }

echo "== native binary behavior"
if [[ "$(uname -s)" == MINGW* || "$(uname -s)" =~ NT-|Windows ]] || command -v wine >/dev/null 2>&1; then
  BIN_TO_RUN="${DIST}/winzip/dbguard.exe"
else
  # On non-Windows CI hosts there may be no interpreter for PE binaries.
  BIN_TO_RUN=""
  echo "(skipping windows exe execution on $(uname -s))"
fi
if [[ -n "${BIN_TO_RUN}" ]]; then
  VERSION_OUT="$("${BIN_TO_RUN}" version)"
  [[ "${VERSION_OUT}" == *"dbguard version ${VERSION}"* ]] || { echo "unexpected version output: ${VERSION_OUT}" >&2; exit 1; }
  MODE_JSON="$("${BIN_TO_RUN}" review-sql 'SELECT 1;' --format json)"
  echo "${MODE_JSON}" | grep -q '"coverage_mode": *"core"' || { echo "portable artifact did not report core mode: ${MODE_JSON}" >&2; exit 1; }
  set +e
  "${BIN_TO_RUN}" review-sql 'DELETE FROM users;' --format json >/dev/null
  code=$?
  set -e
  [[ "${code}" == "3" ]] || { echo "expected block exit 3 from portable artifact, got ${code}" >&2; exit 1; }
fi

if [[ -n "${DBWALL_SMOKE_FULL_BIN:-}" ]]; then
  echo "== packaging provided full-mode binary like release.yml"
  FULL_OUTDIR="${DIST}/dbguard_${VERSION}_linux_amd64_full"
  mkdir -p "${FULL_OUTDIR}"
  cp "${DBWALL_SMOKE_FULL_BIN}" "${FULL_OUTDIR}/dbguard"
  cp README.md LICENSE "${FULL_OUTDIR}/"
  printf '%s\n' "coverage_mode=full" "cgo_enabled=1" > "${FULL_OUTDIR}/COVERAGE_MODE.txt"
  tar -C "${FULL_OUTDIR}" -czf "${DIST}/dbguard_${VERSION}_linux_amd64_full.tar.gz" .
  rm -rf "${FULL_OUTDIR}"
  (cd "${DIST}" && sha256sum -- *.tar.gz *.zip > checksums.txt)
  normalize_tar_list "${DIST}/dbguard_${VERSION}_linux_amd64_full.tar.gz" | grep -qx 'COVERAGE_MODE.txt' \
    || { echo "full archive missing COVERAGE_MODE.txt" >&2; exit 1; }
fi

echo "== artifacts"
ls -1 "${DIST}"
echo "release smoke OK"
