#!/usr/bin/env bash
# Offline verification of the checksum-selection logic used by action.yml.
# Mirrors the awk/hash-validation snippet so a regression in the action's
# asset matching is caught in CI without downloading any release artifact.
set -euo pipefail

select_hash() {
  local asset="$1" file="$2"
  awk -v asset="${asset}" '$2==asset || $2=="./"asset {print $1}' "${file}" | head -n1
}

assert_select() {
  local want="$1" got="$2" label="$3"
  if [[ "${want}" != "${got}" ]]; then
    echo "FAIL ${label}: want=${want} got=${got}" >&2
    exit 1
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat >"${tmp}/checksums.txt" <<'EOF'
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  dbguard_v9.9.9_linux_amd64_full.tar.gz
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  dbguard_v1.2.3_linux_amd64_full.tar.gz
cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc  ./dbguard_v1.2.3_windows_amd64.zip
dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  dbguard_v1.2.3_darwin_arm64.tar.gz
EOF

ASSET="dbguard_v1.2.3_linux_amd64_full.tar.gz"
assert_select "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" \
  "$(select_hash "${ASSET}" "${tmp}/checksums.txt")" "bare-name entry"

assert_select "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" \
  "$(select_hash "dbguard_v1.2.3_windows_amd64.zip" "${tmp}/checksums.txt")" "./-prefixed entry"

if [[ -n "$(select_hash "dbguard_v1.2.3_missing.tar.gz" "${tmp}/checksums.txt")" ]]; then
  echo "FAIL missing-asset: selection must be empty" >&2
  exit 1
fi

# A lookalike name must not match through unescaped regex metacharacters.
cat >"${tmp}/lookalike.txt" <<'EOF'
eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee  dbguard_vX1Y2Z3_linux_amd64_full_tar_gz
ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff  dbguard_v1.2.3_linux_amd64_full.tar.gz
EOF
assert_select "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" \
  "$(select_hash "dbguard_v1.2.3_linux_amd64_full.tar.gz" "${tmp}/lookalike.txt")" "exact match beats lookalike"

echo "action checksum match checks OK"
