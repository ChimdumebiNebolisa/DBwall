# Generic CI Example

This example works in generic shell-based CI systems where you want JSON output for machine decisions.

```bash
#!/usr/bin/env bash
set -euo pipefail

DBWALL_VERSION="v0.2.0"
curl -L -o dbwall.tar.gz "https://github.com/ChimdumebiNebolisa/DBwall/releases/download/${DBWALL_VERSION}/dbguard_${DBWALL_VERSION}_linux_amd64.tar.gz"
tar -xzf dbwall.tar.gz
chmod +x dbguard

./dbguard review-file ./migrations/latest.sql --policy ./migrations/dbguard.yaml --format json > dbwall.json
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

If you need full PostgreSQL parser-backed coverage in CI, replace the binary download with a source build using `CGO_ENABLED=1`.
