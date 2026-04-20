# Using DBwall in GitHub Actions

This workflow example downloads a tagged DBwall binary, scans changed SQL, uploads SARIF to GitHub code scanning, and fails the job on blocking findings.

```yaml
name: dbwall

on:
  pull_request:
    paths:
      - "migrations/**/*.sql"
      - "scripts/**/*.sql"
  push:
    branches: [main]
    paths:
      - "migrations/**/*.sql"
      - "scripts/**/*.sql"

jobs:
  review-sql:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download DBwall release binary
        run: |
          VERSION="v0.2.0"
          curl -L -o dbwall.tar.gz "https://github.com/ChimdumebiNebolisa/DBwall/releases/download/${VERSION}/dbguard_${VERSION}_linux_amd64.tar.gz"
          tar -xzf dbwall.tar.gz
          chmod +x dbguard

      - name: Review SQL and emit SARIF
        run: |
          ./dbguard review-file ./migrations/latest.sql --policy ./migrations/dbguard.yaml --format sarif > dbwall.sarif
          ./dbguard review-file ./migrations/latest.sql --policy ./migrations/dbguard.yaml
        continue-on-error: true

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: dbwall.sarif

      - name: Fail on block
        run: |
          ./dbguard review-file ./migrations/latest.sql --policy ./migrations/dbguard.yaml --format json > dbwall.json
          python - <<'PY'
          import json
          with open("dbwall.json", "r", encoding="utf-8") as fh:
              data = json.load(fh)
          if data["decision"] == "block":
              raise SystemExit(3)
          PY
```

Notes:
- Tagged release binaries are the easiest install path for CI.
- Release binaries are built in core coverage mode. For full PostgreSQL parser-backed coverage, build DBwall from source with `CGO_ENABLED=1`.
- Use `--format json` for machine decisions and `--format sarif` for code scanning integrations.
