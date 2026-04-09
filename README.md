# DBwall

**Go CLI that reviews AI-generated PostgreSQL SQL and blocks unsafe operations before execution.**

dbguard checks generated SQL before it reaches a database. It focuses on a small set of high-risk PostgreSQL operations such as full-table deletes, full-table updates, `DROP TABLE`, `ALTER TABLE ... DROP COLUMN`, and writes against protected tables. The CLI returns a simple `allow`, `warn`, or `block` decision that works cleanly in local workflows and CI.

## Why this exists

AI-generated SQL often looks plausible while still being destructive. A missing `WHERE` clause or schema-changing statement can cause real data loss. dbguard adds a fast local review step before execution or merge.

## Parsing behavior

- With `CGO_ENABLED=1`, dbguard uses `pg_query_go` for PostgreSQL parser-backed analysis.
- Without CGO, dbguard uses a built-in cross-platform parser that covers the current v1 rule set on Windows, macOS, and Linux.
- The supported v1 checks are exercised by the automated test suite in both CLI and parser-level tests.

## v1 features

| Feature | Description |
| --- | --- |
| `DELETE` without `WHERE` | Block by default |
| `UPDATE` without `WHERE` | Block by default |
| `DROP TABLE` | Block by default |
| `ALTER TABLE ... DROP COLUMN` | Block by default |
| Writes to protected tables | Warn by default |
| Decisions | `allow`, `warn`, `block` |
| Exit codes | `0` allow, `1` error, `2` warn, `3` block |

## Installation

Requires **Go 1.21+**.

```bash
git clone https://github.com/ChimdumebiNebolisa/DBwall.git
cd DBwall
go mod tidy
go build -o dbguard ./cmd/dbguard
```

If you want the PostgreSQL parser-backed path, build with `CGO_ENABLED=1` and a working C toolchain.

On Windows the binary will be `dbguard.exe`; use `.\dbguard.exe` in examples below if needed.

## Usage

```bash
dbguard review-sql "DELETE FROM users;"
dbguard review-sql "SELECT 1;" --format json
dbguard review-file ./examples/protected_table_update.sql --policy ./examples/dbguard.yaml
dbguard version
```

### Expected decisions

`DELETE FROM users;` returns `BLOCK` with exit code `3`.

`SELECT 1; --format json` returns `ALLOW` with exit code `0`.

`UPDATE users SET role = 'viewer' WHERE id = 1;` with `users` in `protected_tables` returns `WARN` with exit code `2`.

## Policy file

If no policy file is provided, built-in defaults are used.

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
| --- | --- |
| `0` | Allow |
| `1` | Internal or tool error |
| `2` | Warn |
| `3` | Block |

## Limitations

- PostgreSQL-focused only.
- No semantic predicate analysis. `DELETE FROM users WHERE 1=1` still counts as having a `WHERE` clause.
- The non-CGO parser is intentionally scoped to the current v1 checks.

## Local development

```bash
go build ./cmd/dbguard
go test ./...
go fmt ./...
go vet ./...
```

## CI

This repo includes CI for build and test coverage on pushes and pull requests. See [.github/workflows/ci.yml](.github/workflows/ci.yml).

For a GitHub Actions usage example, see [examples/GITHUB_ACTION_EXAMPLE.md](examples/GITHUB_ACTION_EXAMPLE.md).

## License

[MIT](LICENSE)
