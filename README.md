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

## What DBwall Does Not Do

DBwall is a SQL review gate, not a database firewall or a substitute for database permissions.

- It does not execute queries or enforce runtime access control.
- It does not claim full semantic understanding of every PostgreSQL migration pattern.
- It does not support non-PostgreSQL dialects.
- In `core` coverage mode it deliberately keeps reduced advanced-rule coverage rather than pretending parity with the parser-backed path.

Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | Allow |
| `1` | Tool or parse error |
| `2` | Warn |
| `3` | Block |

## Coverage Modes

DBwall reports its parser coverage mode explicitly:

- `full` (`CGO_ENABLED=1`): security metadata is derived by walking the structured PostgreSQL AST from `pg_query_go` (relations, grants, predicates, nested CTE/subquery sources). JSON/SARIF also expose per-statement `completeness`.
- `core` (`CGO_ENABLED=0`): portable token parser with explicitly reduced semantic coverage. Multi-table joins, nested CTE mutations, and several GRANT/DROP multi-object shapes are only accurate in `full` mode.

Checks that need `full` mode for accurate relation discovery include:

- join / `UPDATE ... FROM` / `DELETE ... USING` secondary relations
- CTE and subquery sources (including `COPY (SELECT ...)`)
- multi-object `DROP TABLE` / `TRUNCATE` / `GRANT ... ON TABLE a, b`
- constant-folded trivial predicates beyond `TRUE` and `1 = 1`
- explicit `semantic_analysis_incomplete` findings for unsupported AST statement types

Portable release archives are built with `CGO_ENABLED=0` (`core` mode). Tagged releases also publish a Linux amd64 **full-mode** archive (`dbguard_<tag>_linux_amd64_full.tar.gz`, `CGO_ENABLED=1`) for first-party PR gates.

## Install

### Build from source

This is the reliable install path for the current repo state.

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

### Tagged release binaries

Release archives are produced by [.github/workflows/release.yml](.github/workflows/release.yml) when a semver tag such as `v0.2.0` is pushed. Use the GitHub Releases page for the currently published version instead of hardcoding a version string from the README.

## Quick Start

Build the CLI and review one statement:

```bash
go build -o dbguard ./cmd/dbguard
./dbguard review-sql "DELETE FROM users;"
```

Review a file with policy and machine-readable output:

```bash
./dbguard review-file ./migrations/latest.sql --policy ./examples/dbguard.yaml --format json
```

Review every changed SQL file in one aggregated decision (PR-gate style):

```bash
./dbguard review-files ./migrations/001.sql ./migrations/002.sql --policy ./examples/dbguard.yaml --format sarif > dbwall.sarif
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

Multi-file SARIF (per-file locations, one aggregated decision):

```bash
dbguard review-files ./a.sql ./b.sql --policy ./dbguard.yaml --format sarif > dbwall.sarif
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

- Reusable GitHub Action: [`action.yml`](action.yml) (inputs: `version`, `policy`, `sql-paths`, `fail-on-warn`)
- GitHub Actions example (changed `.sql` PR gate): [examples/GITHUB_ACTION_EXAMPLE.md](examples/GITHUB_ACTION_EXAMPLE.md)
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
- multi-table / CTE / COPY-select / multi-object and incomplete-analysis cases (many require `full` mode)

## Benchmark

The reproducible benchmark harness lives under `benchmark/`.

Run it from the repo root:

```bash
CGO_ENABLED=1 go run ./benchmark/cmd/dbwallbench --repo-root . --manifest ./benchmark/manifest.json --json-out ./benchmark/results/benchmark_results.json --report-out ./benchmark/reports/benchmark_report.md
```

Saved artifacts:

- Raw benchmark results: [benchmark/results/benchmark_results.json](benchmark/results/benchmark_results.json)
- Human-readable report: [benchmark/reports/benchmark_report.md](benchmark/reports/benchmark_report.md)

Current saved run from [benchmark/results/benchmark_results.json](benchmark/results/benchmark_results.json):

- Corpus: `benchmark/manifest.json`
- Coverage mode: `full`
- Total cases: `29`
- Correct blocks: `13`
- Correct allows: `6`
- Correct warns: `10`
- False positives: `0`
- False negatives: `0`
- Precision (`block` as positive class): `1.0000`
- Recall (`block` as positive class): `1.0000`
- Accuracy (exact decision match): `1.0000`
- Average runtime per case: `4.157 ms`

Those numbers are measured results from the saved artifact, not a generalized product claim. Precision and recall use `block` as the positive class. Cases marked `requires_full` are included only when the built binary reports `coverage_mode=full`.

## Local Development

```bash
CGO_ENABLED=1 go test ./...
CGO_ENABLED=0 go test ./...
go vet ./...
```

## License

[MIT](LICENSE)
