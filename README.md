# dbguard

**AST-based Go CLI that reviews AI-generated PostgreSQL SQL and blocks unsafe operations before execution.**

AI agents and code assistants increasingly generate SQL and migration snippets. Those outputs can be dangerous. dbguard parses PostgreSQL SQL using a real parser, applies configurable safety rules, and returns an allow/warn/block decision. Built for developers, CI pipelines, and agent toolchains.

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
- Parse with a real PostgreSQL parser
- Apply v1 safety rules (see [docs/SPEC.md](docs/SPEC.md))
- Load configurable policy from YAML
- Human-readable and JSON reports
- Exit codes suitable for CI

## Installation

```bash
go build -o dbguard ./cmd/dbguard
```

## Usage

- Review inline SQL: `dbguard review-sql "DELETE FROM users;"`
- Review a file: `dbguard review-file ./examples/delete_all.sql`
- With policy: `dbguard review-file ./examples/delete_all.sql --policy ./examples/dbguard.yaml`
- JSON output: `dbguard review-sql "DROP TABLE users;" --format json`
- Version: `dbguard version`

## Exit codes

- `0` = allow, no violations
- `1` = internal/tool error
- `2` = warn decision
- `3` = block decision

## Local development

```bash
go build ./cmd/dbguard
go test ./...
```

## Documentation

- [SPEC.md](docs/SPEC.md) – Problem, scope, rules, CLI contract
- [MILESTONES.md](docs/MILESTONES.md) – Progress and substeps
- [GUARDRAILS.md](docs/GUARDRAILS.md) – Engineering guardrails
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) – Code structure and data flow

## License

See repository license.
