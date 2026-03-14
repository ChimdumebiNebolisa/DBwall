# dbguard – Specification

## Problem statement

AI-generated SQL and migrations can perform unsafe operations (full-table DELETE/UPDATE, DROP TABLE, DROP COLUMN, writes to protected tables). Executing such statements without review risks data loss and compliance violations. A tool is needed to parse SQL, apply safety rules, and return a clear allow/warn/block decision before execution.

## Scope (v1)

- **Dialect:** PostgreSQL only.
- **Input:** SQL string or file.
- **Analysis:** AST-based; no regex as the core enforcement engine.
- **Output:** Human and JSON reports; exit codes for CI.
- **Policy:** YAML config for rules and protected tables.

## Non-goals (v1)

- Multi-dialect support
- ALTER type narrowing, full-table lock detection, missing index detection
- Query plan analysis, semantic row-count estimation
- Database connection / live schema introspection
- Generic agent plan JSON ingestion, remote API, web UI, LLM integration
- Autonomous approval flows

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md).

## Rule definitions

| Rule | Trigger | Default decision | Default severity |
|------|--------|------------------|------------------|
| delete_without_where | DELETE with no WHERE | block | critical |
| update_without_where | UPDATE with no WHERE | block | critical |
| drop_table | DROP TABLE | block | critical |
| drop_column | ALTER TABLE ... DROP COLUMN | block | high/critical |
| writes_to_protected_tables | Write to protected table | warn | medium/high |

## CLI contract

- `dbguard review-sql "<sql>"` – review inline SQL
- `dbguard review-file <path>` – review file
- `--policy <path>` – optional policy file
- `--format human|json` – output format
- `dbguard version` – print version

## Exit code contract

- `0` – allow, no violations
- `1` – internal/tool error
- `2` – warn decision
- `3` – block decision

## Policy schema

- `dialect`: must be `postgres` in v1
- `protected_tables`: list of table names (exact match in v1)
- `rules`: map of rule name to `allow` | `warn` | `block`

See examples in repo.

## Known limitations

- **PostgreSQL only**; no semantic analysis of predicates (e.g. `WHERE 1=1` is treated as having a WHERE clause and does not trigger delete_without_where).
- **Parser:** Uses pg_query_go (PostgreSQL parser); requires CGO for build; first build can take several minutes. Multi-statement input is parsed in order; each statement is analyzed independently.
- **Table names:** Extracted from the parse tree; qualified names (schema.table) are normalized to the table part where applicable.
