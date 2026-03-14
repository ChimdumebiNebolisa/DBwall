# Using dbguard in GitHub Actions

This page shows how to run dbguard in your CI to block unsafe SQL before merge or deploy.

## Overview

- **Check out** your repo (with SQL migrations or ad-hoc scripts).
- **Build dbguard** from source (Go + CGO; Ubuntu has `gcc` by default).
- **Run** `dbguard review-file` on your SQL file(s).
- **Fail the job** when dbguard returns exit code **3** (block). Optionally fail on **2** (warn).

## Example: review a migration on every PR

Add a workflow (e.g. `.github/workflows/dbguard.yml`) in the repo that contains your SQL:

```yaml
name: dbguard

on:
  pull_request:
    paths:
      - 'migrations/**/*.sql'
      - 'scripts/**/*.sql'
  push:
    branches: [main]
    paths:
      - 'migrations/**/*.sql'
      - 'scripts/**/*.sql'

jobs:
  review-sql:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repo
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.21"

      - name: Clone and build dbguard
        env:
          CGO_ENABLED: 1
        run: |
          git clone --depth 1 https://github.com/ChimdumebiNebolisa/DBwall.git /tmp/dbguard
          cd /tmp/dbguard
          go mod tidy
          go build -o dbguard ./cmd/dbguard

      - name: Review SQL (fail on block)
        run: |
          /tmp/dbguard/dbguard review-file ./migrations/latest.sql --policy ./migrations/dbguard.yaml
          exitcode=$?
          if [ $exitcode -eq 3 ]; then
            echo "::error::dbguard blocked: unsafe SQL (exit 3)"
            exit 3
          fi
          if [ $exitcode -eq 2 ]; then
            echo "::warning::dbguard reported warnings (exit 2)"
            # exit 2  # uncomment to fail on warn
          fi
          exit 0
        working-directory: ${{ github.workspace }}
```

Adjust paths (`migrations/latest.sql`, `migrations/dbguard.yaml`) to match your repo. If you don't use a policy file, omit `--policy ./migrations/dbguard.yaml`.

## Simpler: single file, fail on block only

```yaml
- name: Review migration
  run: |
    cd /tmp/dbguard && CGO_ENABLED=1 go build -o dbguard ./cmd/dbguard
    /tmp/dbguard/dbguard review-file ${{ github.workspace }}/path/to/migration.sql
    if [ $? -eq 3 ]; then exit 3; fi
```

## Exit codes (reminder)

| Code | Meaning |
|------|--------|
| 0 | Allow |
| 1 | Tool error |
| 2 | Warn |
| 3 | Block — use this to fail the job |

## Optional: surface output in the Actions log

The default human output is already printed to stdout. For JSON in logs you can run with `--format json` and optionally `jq` for formatting.

## This repo's own CI

The dbguard project's CI (build + test) is in [.github/workflows/ci.yml](../.github/workflows/ci.yml). It does not run dbguard against example SQL; it only builds and runs unit tests.
