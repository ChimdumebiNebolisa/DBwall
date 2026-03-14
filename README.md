# dbguard

**AST-based Go CLI that reviews AI-generated PostgreSQL SQL and blocks unsafe operations before execution.**

AI agents and code assistants increasingly generate SQL and migration snippets. Those outputs can be dangerous. dbguard parses PostgreSQL SQL using a real parser, applies configurable safety rules, and returns an allow/warn/block decision. Built for developers, CI pipelines, and agent toolchains.

## Why guardrails for AI-generated SQL

AI-generated SQL can include destructive or risky operations: `DELETE FROM users;`, `UPDATE payments SET ...;` without a WHERE clause, `DROP TABLE`, or `ALTER TABLE ... DROP COLUMN`. Running such statements without review risks data loss and compliance issues. dbguard gives you a fast, local check before execution or before merging migrations.

## What dbguard is

- A local CLI
- PostgreSQL-only (v1)
- AST-based, not regex-based
- Focused on a small, credible rule set
- Designed for reliability and strong resume value

## What dbguard is not

- A chatbot, multi-agent framework, or cloud SaaS
- A database proxy, generic SQL linter, or query planner
- A full migration engine

## v1 features

- Accept PostgreSQL SQL from string or file
- Parse with a real PostgreSQL parser (pg_query_go)
- Apply v1 safety rules: DELETE/UPDATE without WHERE, DROP TABLE, ALTER TABLE DROP COLUMN, writes to protected tables
- Load configurable policy from YAML
- Human-readable and JSON reports
- Exit codes suitable for CI

## Installation

Requires Go 1.21+ and CGO (for the PostgreSQL parser). First build may take a few minutes.

```bash
go build -o dbguard ./cmd/dbguard
```

## Usage

**Review inline SQL**

```bash
dbguard review-sql "DELETE FROM users;"
```

**Review a SQL file**

```bash
dbguard review-file ./examples/delete_all.sql
```

**With a policy file**

```bash
dbguard review-file ./examples/delete_all.sql --policy ./examples/dbguard.yaml
```

**JSON output (for CI or pipelines)**

```bash
dbguard review-sql "DROP TABLE users;" --format json
```

**Version**

```bash
dbguard version
```

## Policy file

If no policy file is provided, built-in defaults are used. Example `dbguard.yaml`:

```yaml
dialect: postgres

protected_tables:
  - users
  - payments
  - audit_logs

rules:
  delete_without_where: block
  update_without_where: block
  drop_table: block
  drop_column: block
  writes_to_protected_tables: warn
```

See [examples/dbguard.yaml](examples/dbguard.yaml).

## Exit codes

| Code | Meaning |
|------|--------|
| 0 | Allow – no violations |
| 1 | Internal or tool error |
| 2 | Warn – at least one warning |
| 3 | Block – at least one blocking violation |

Use in CI: `dbguard review-file migration.sql; exitcode=$?; [ $exitcode -eq 0 ] || [ $exitcode -eq 2 ]` to allow or warn but fail on block.

## Limitations (v1)

- **PostgreSQL only.** No other dialects.
- **No semantic predicate analysis.** e.g. `DELETE FROM users WHERE 1=1` has a WHERE clause and does not trigger `delete_without_where`.
- **Parser:** Build requires CGO; first compile can take several minutes (PostgreSQL parser).

See [docs/SPEC.md](docs/SPEC.md) for full specification and limitations.

## Local development

```bash
go build ./cmd/dbguard
go test ./...
```

Format and vet:

```bash
go fmt ./...
go vet ./...
```

## CI

GitHub Actions runs on push/PR to `main`: build and test with CGO enabled. See [.github/workflows/ci.yml](.github/workflows/ci.yml). First run may be slow due to parser build.

## Documentation

- [docs/SPEC.md](docs/SPEC.md) – Problem, scope, rules, CLI contract, policy schema
- [docs/MILESTONES.md](docs/MILESTONES.md) – Progress and substeps
- [docs/GUARDRAILS.md](docs/GUARDRAILS.md) – Engineering guardrails
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) – Code structure and data flow

## License

See repository license.
