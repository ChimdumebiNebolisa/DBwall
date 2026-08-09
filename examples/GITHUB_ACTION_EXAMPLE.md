# Using DBwall in GitHub Actions

This workflow uses the first-party composite action to download the **full-mode** Linux binary, review every changed `.sql` file on a pull request, upload SARIF to GitHub code scanning, and fail the job on blocking findings.

```yaml
name: dbwall

on:
  pull_request:
    paths:
      - "**/*.sql"
      - "**/dbguard.yaml"
  push:
    branches: [main]
    paths:
      - "**/*.sql"
      - "**/dbguard.yaml"

permissions:
  contents: read
  pull-requests: read
  security-events: write

jobs:
  review-sql:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Collect changed SQL files
        id: changed
        shell: bash
        run: |
          set -euo pipefail
          if [[ "${{ github.event_name }}" == "pull_request" ]]; then
            BASE="${{ github.event.pull_request.base.sha }}"
            HEAD="${{ github.event.pull_request.head.sha }}"
          else
            BASE="${{ github.event.before }}"
            HEAD="${{ github.sha }}"
          fi
          if [[ -z "${BASE}" || "${BASE}" == "0000000000000000000000000000000000000000" ]]; then
            mapfile -t files < <(git ls-files '*.sql')
          else
            mapfile -t files < <(git diff --name-only --diff-filter=ACMR "${BASE}" "${HEAD}" -- '*.sql')
          fi
          {
            echo "sql-paths<<EOF"
            printf '%s\n' "${files[@]}"
            echo "EOF"
          } >> "${GITHUB_OUTPUT}"

      - name: Review changed SQL with DBwall
        id: dbwall
        uses: ChimdumebiNebolisa/DBwall@v0.2.0
        with:
          version: v0.2.0
          policy: ./examples/dbguard.yaml
          sql-paths: ${{ steps.changed.outputs.sql-paths }}
          fail-on-warn: "false"
          sarif-file: dbwall.sarif

      - name: Upload SARIF
        if: always() && hashFiles('dbwall.sarif') != ''
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: dbwall.sarif
```

Notes:
- The reusable action downloads `dbguard_<tag>_linux_amd64_full.tar.gz` (built with `CGO_ENABLED=1`) so multi-object `GRANT`/`DROP`/`TRUNCATE`, joins, and CTE sources are reviewed in `full` coverage mode.
- Pass every changed `.sql` path through `sql-paths`; the CLI aggregates findings across files into one decision (strictest wins) and emits SARIF with per-file locations.
- Set `fail-on-warn: "true"` to fail the job on warn findings as well as blocks.
- Portable core-mode archives (`linux_amd64` without the `_full` suffix) remain available for non-Linux or no-CGO installs, but they are not what this PR gate uses.
