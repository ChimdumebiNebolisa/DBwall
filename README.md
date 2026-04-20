# DBwall

**PostgreSQL-first SQL security gate for AI-generated queries, migrations, and automation workflows.**

DBwall reviews PostgreSQL SQL before it reaches a database or merge target. It is intentionally PostgreSQL-only and focuses on high-risk statements: destructive DDL/DML, risky privilege changes, and suspicious bulk-access patterns. The repository is named `DBwall`; the CLI binary is `dbguard`.

## Why DBwall

AI-generated SQL is often syntactically valid but operationally unsafe. DBwall is meant to catch statements such as:

- unbounded `DELETE` and `UPDATE`
- destructive schema operations like `DROP TABLE`, `DROP SCHEMA`, and safety-boundary removal
- privilege expansion such as `GRANT ... TO PUBLIC`
- suspicious reads or exports from protected objects

It returns one of three decisions:

- `allow`
- `warn`
- `block`

Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | Allow |
| `1` | Tool or parse error |
| `2` | Warn |
| `3` | Block |

## Coverage Modes

DBwall reports its parser coverage mode explicitly:

- `full`: PostgreSQL parser-backed validation when built with `CGO_ENABLED=1`
- `core`: portable fallback mode with reduced advanced-rule coverage

Release binaries are built for portability, so they run in `core` mode unless you build from source with CGO enabled.

## Install

### Recommended: release binary

Download a tagged release from GitHub Releases and run the binary directly.

Linux:

```bash
curl -L -o dbwall.tar.gz https://github.com/ChimdumebiNebolisa/DBwall/releases/download/v0.2.0/dbguard_v0.2.0_linux_amd64.tar.gz
tar -xzf dbwall.tar.gz
./dbguard version
```

Windows PowerShell:

```powershell
Invoke-WebRequest -Uri https://github.com/ChimdumebiNebolisa/DBwall/releases/download/v0.2.0/dbguard_v0.2.0_windows_amd64.zip -OutFile dbwall.zip
Expand-Archive dbwall.zip -DestinationPath .
.\dbguard.exe version
```

### Build from source

Portable build:

```bash
git clone https://github.com/ChimdumebiNebolisa/DBwall.git
cd DBwall
go build -o dbguard ./cmd/dbguard
```

Full PostgreSQL parser-backed build:

```bash
CGO_ENABLED=1 go build -o dbguard ./cmd/dbguard
```

### Go install

```bash
go install github.com/ChimdumebiNebolisa/DBwall/cmd/dbguard@latest
```

## Usage

Human-readable review:

```bash
dbguard review-sql "DELETE FROM users;"
```

JSON for automation:

```bash
dbguard review-file ./migrations/latest.sql --policy ./dbguard.yaml --format json
```

SARIF for code scanning:

```bash
dbguard review-file ./migrations/latest.sql --policy ./dbguard.yaml --format sarif > dbwall.sarif
```

Version:

```bash
dbguard version
```

## Output Modes

- `human`: concise summary, per-statement findings, rationale, remediation, and coverage-mode note
- `json`: stable machine-readable output with decision, severity, summary, tool/version metadata, and finding details
- `sarif`: code-scanning output for GitHub and similar tooling

## Policy

DBwall stays additive and PostgreSQL-specific. The policy file supports:

- `dialect`
- `protected_tables`
- `protected_schemas`
- `protected_roles`
- per-rule `allow|warn|block` overrides in `rules`

Example:

```yaml
dialect: postgres

protected_tables:
  - users
  - payments

protected_schemas:
  - finance

protected_roles:
  - pg_read_all_data

rules:
  delete_without_where: block
  truncate_table: block
  writes_to_protected_tables: warn
  select_without_limit_from_protected_table: warn
```

Full example: [examples/dbguard.yaml](examples/dbguard.yaml)

## Integrations

- GitHub Actions: [examples/GITHUB_ACTION_EXAMPLE.md](examples/GITHUB_ACTION_EXAMPLE.md)
- Pre-commit: [examples/PRE_COMMIT_EXAMPLE.md](examples/PRE_COMMIT_EXAMPLE.md)
- Generic CI: [examples/CI_EXAMPLE.md](examples/CI_EXAMPLE.md)

Repo workflows:

- CI: [.github/workflows/ci.yml](.github/workflows/ci.yml)
- Release: [.github/workflows/release.yml](.github/workflows/release.yml)

## Test Corpus

DBwall includes an adversarial corpus under [test_e2e/testdata/corpus.json](test_e2e/testdata/corpus.json) covering:

- good queries
- borderline queries
- obviously dangerous queries
- false-positive cases

## Benchmark

The reproducible benchmark harness lives under `benchmark/`.

Run it from the repo root:

```bash
go run ./benchmark/cmd/dbwallbench --repo-root . --manifest ./benchmark/manifest.json --json-out ./benchmark/results/benchmark_results.json --report-out ./benchmark/reports/benchmark_report.md
```

Saved artifacts:

- Raw benchmark results: [benchmark/results/benchmark_results.json](benchmark/results/benchmark_results.json)
- Human-readable report: [benchmark/reports/benchmark_report.md](benchmark/reports/benchmark_report.md)

Benchmark summaries should be taken from those saved artifacts, not from README text.

## Local Development

```bash
go test ./...
go vet ./...
```

## License

[MIT](LICENSE)
