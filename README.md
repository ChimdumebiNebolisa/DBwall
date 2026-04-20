# DBwall

**PostgreSQL-first SQL security gate for AI-generated queries, migrations, and workflow automation.**

DBwall reviews PostgreSQL SQL before it reaches a database or merge target. It stays intentionally PostgreSQL-focused and blocks or warns on destructive DDL/DML, risky permission changes, and suspicious bulk-access patterns. It is designed for local developer workflows, CI, and security tooling integrations.

## What DBwall checks

DBwall keeps a PostgreSQL-only rule set instead of diluting coverage across multiple dialects.

Current rule categories:

- Destructive DML: `delete_without_where`, `delete_trivial_where`, `update_without_where`, `update_trivial_where`, `truncate_table`
- Destructive DDL: `drop_table`, `drop_schema`, `drop_database`, `drop_column`, `alter_table_drop_not_null_or_constraint`
- Permissions: `grant_to_public_on_protected_objects`, `alter_default_privileges_public`, `grant_high_risk_role_membership`
- Protected object handling: `writes_to_protected_tables`
- Bulk access: `select_all_from_protected_table`, `select_without_limit_from_protected_table`, `copy_to_stdout_or_program_from_protected_source`

## Coverage modes

DBwall has two explicit coverage modes:

- `full`: parser-backed PostgreSQL validation when built with `CGO_ENABLED=1`
- `core`: portable fallback mode with reduced advanced-rule coverage

DBwall reports the current coverage mode in JSON output and in human-readable summaries. Release binaries are built in `core` mode for portability. If you want full PostgreSQL parser-backed coverage, build from source with CGO enabled.

## Install

### Recommended: tagged release binaries

Download a tagged release from GitHub Releases and run the binary directly.

Linux:

```bash
curl -L -o dbwall.tar.gz https://github.com/ChimdumebiNebolisa/DBwall/releases/download/v0.2.0/dbguard_v0.2.0_linux_amd64.tar.gz
tar -xzf dbwall.tar.gz
./dbguard version
```

macOS:

```bash
curl -L -o dbwall.tar.gz https://github.com/ChimdumebiNebolisa/DBwall/releases/download/v0.2.0/dbguard_v0.2.0_darwin_arm64.tar.gz
tar -xzf dbwall.tar.gz
./dbguard version
```

Windows PowerShell:

```powershell
Invoke-WebRequest -Uri https://github.com/ChimdumebiNebolisa/DBwall/releases/download/v0.2.0/dbguard_v0.2.0_windows_amd64.zip -OutFile dbwall.zip
Expand-Archive dbwall.zip -DestinationPath .
.\dbguard.exe version
```

### Source build

Portable core-mode build:

```bash
git clone https://github.com/ChimdumebiNebolisa/DBwall.git
cd DBwall
go build -o dbguard ./cmd/dbguard
```

Full parser-backed build:

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

Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | Allow |
| `1` | Tool or parse error |
| `2` | Warn |
| `3` | Block |

## Output modes

- `human`: summary counts, per-statement findings, rationale, remediation, and coverage-mode note
- `json`: stable machine-readable output with legacy fields preserved and extended metadata including `tool`, `version`, `summary`, `generated_at`, and `coverage_mode`
- `sarif`: code-scanning oriented output suitable for GitHub and similar tooling

## Policy file

Policy remains additive and PostgreSQL-specific:

```yaml
dialect: postgres

protected_tables:
  - users
  - payments
  - audit_logs

protected_schemas:
  - finance
  - admin

protected_roles:
  - pg_read_all_data
  - platform_admin

rules:
  delete_without_where: block
  delete_trivial_where: block
  update_without_where: block
  update_trivial_where: block
  truncate_table: block
  drop_table: block
  drop_schema: block
  drop_database: block
  drop_column: block
  alter_table_drop_not_null_or_constraint: block
  writes_to_protected_tables: warn
  grant_to_public_on_protected_objects: block
  alter_default_privileges_public: block
  grant_high_risk_role_membership: block
  select_all_from_protected_table: warn
  select_without_limit_from_protected_table: warn
  copy_to_stdout_or_program_from_protected_source: block
```

See [examples/dbguard.yaml](examples/dbguard.yaml).

## Workflow integration

- GitHub Actions example: [examples/GITHUB_ACTION_EXAMPLE.md](examples/GITHUB_ACTION_EXAMPLE.md)
- Pre-commit example: [examples/PRE_COMMIT_EXAMPLE.md](examples/PRE_COMMIT_EXAMPLE.md)
- Generic CI example: [examples/CI_EXAMPLE.md](examples/CI_EXAMPLE.md)

## Release and CI story

- CI validates both `full` and `core` modes where practical and runs `go vet`, `staticcheck`, and `govulncheck`
- Tagged releases build Linux, macOS, and Windows archives with checksums
- Release builds inject version metadata through `ldflags`

Relevant workflows:

- [ci.yml](.github/workflows/ci.yml)
- [release.yml](.github/workflows/release.yml)

## Test corpus

DBwall includes an adversarial corpus under `test_e2e/testdata/corpus.json` with:

- good queries
- borderline queries
- obviously dangerous queries
- false-positive cases

The corpus is exercised in automated tests alongside parser, rules, report, and CLI end-to-end coverage.

## Local development

```bash
go test ./...
go vet ./...
```

## License

[MIT](LICENSE)
