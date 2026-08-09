# Generic CI Example

Use the published **full-mode** Linux artifact and `review-files` so multi-object
`GRANT`/`DROP`/`TRUNCATE`, joins, and CTE sources are reviewed with
`coverage_mode=full`.

**Requirement:** tag `v0.2.0` (or newer) must publish
`dbguard_<tag>_linux_amd64_full.tar.gz` together with `checksums.txt`. Older tags
that only ship the portable core archive are not sufficient for this gate.

```bash
#!/usr/bin/env bash
set -euo pipefail

DBWALL_VERSION="v0.2.0"
ASSET="dbguard_${DBWALL_VERSION}_linux_amd64_full.tar.gz"
BASE_URL="https://github.com/ChimdumebiNebolisa/DBwall/releases/download/${DBWALL_VERSION}"

curl -fsSL -o "${ASSET}" "${BASE_URL}/${ASSET}"
curl -fsSL -o checksums.txt "${BASE_URL}/checksums.txt"
grep -F " ${ASSET}" checksums.txt | sha256sum -c -

tar -xzf "${ASSET}"
chmod +x dbguard

# Discover SQL inputs for the change under review (newline-delimited; spaces OK).
mapfile -t SQL_FILES < <(git diff --name-only --diff-filter=ACMR "${BASE_SHA}" "${HEAD_SHA}" -- '*.sql')
if [[ ${#SQL_FILES[@]} -eq 0 ]]; then
  echo "No SQL files changed; nothing to review."
  exit 0
fi

./dbguard review-files "${SQL_FILES[@]}" \
  --policy ./examples/dbguard.yaml \
  --format json > dbwall.json
cat dbwall.json

python - <<'PY'
import json
with open("dbwall.json", "r", encoding="utf-8") as fh:
    data = json.load(fh)
if data["decision"] == "block":
    raise SystemExit(3)
if data["decision"] == "warn":
    raise SystemExit(2)
PY
```

### Single-file example

If you only need to review one known file (not a PR diff), pass that path to
`review-file` or `review-files`:

```bash
./dbguard review-file ./migrations/latest.sql \
  --policy ./migrations/dbguard.yaml \
  --format json > dbwall.json
```

Portable core-mode archives (`dbguard_<tag>_linux_amd64.tar.gz` without `_full`)
remain available for non-CGO installs, but they are not what the first-party PR
gate expects.
