# dbguard

**AST-based Go CLI that reviews AI-generated PostgreSQL SQL and blocks unsafe operations before execution.**

AI agents and code assistants increasingly generate SQL and migration snippets. Those outputs can be dangerous—full-table deletes, schema-dropping statements, or writes to critical tables. dbguard parses PostgreSQL with a **real parser** (pg_query), applies configurable safety rules, and returns a clear **allow / warn / block** decision. Built for developers, CI pipelines, and agent toolchains.

## Why guardrails for AI-generated SQL

AI-generated SQL often looks correct but can be destructive: `DELETE FROM users;`, `UPDATE payments SET ...;` without a WHERE clause, `DROP TABLE`, or `ALTER TABLE ... DROP COLUMN`. Running such statements without review risks data loss and compliance violations. dbguard gives you a fast, local check **before** execution or before merging migrations—no regex heuristics, no guessing.

## Why PostgreSQL AST parsing matters

dbguard uses the same parser as PostgreSQL. That means statement types, table names, and presence of WHERE clauses are derived from the parse tree, not string matching. You get accurate detection of unsafe patterns without false positives from comments or string literals.

---

## What dbguard is

- A **local CLI** — no cloud, no API keys
- **PostgreSQL-only** (v1)
- **AST-based**, not regex-based
- A small, credible rule set
- CI-friendly exit codes and JSON output

## What dbguard is not

- A chatbot, multi-agent framework, or cloud SaaS
- A database proxy, generic SQL linter, or query planner
- A full migration engine

---

## v1 features

| Feature | Description |
|--------|-------------|
| **DELETE without WHERE** | Block (default) or configurable |
| **UPDATE without WHERE** | Block (default) or configurable |
| **DROP TABLE** | Block (default) or configurable |
| **ALTER TABLE … DROP COLUMN** | Block (default) or configurable |
| **Writes to protected tables** | Warn (default); table list from policy |
| **Decisions** | allow / warn / block with configurable policy |
| **Exit codes** | 0 allow, 1 error, 2 warn, 3 block — for CI |

---

## Installation

Requires **Go 1.21+** and **CGO** (for the PostgreSQL parser). First build may take a few minutes.

```bash
git clone https://github.com/ChimdumebiNebolisa/DBwall.git
cd DBwall
go mod tidy
go build -o dbguard ./cmd/dbguard
```

On Windows the binary will be `dbguard.exe`; use `.\dbguard.exe` in examples below if needed.

---

## Demo

### 1. DELETE without WHERE → blocked

```bash
$ dbguard review-sql "DELETE FROM users;"
```

**Expected output:**

```
Decision: BLOCK
Severity: CRITICAL

Statement 1:
  Type: DELETE
  Table: users
  Triggered Rules:
    - delete_without_where
  Reason:
    - DELETE statement has no WHERE clause
  Recommendation:
    - Add a restricting predicate or require manual approval
```

**Exit code:** `3`

---

### 2. DROP TABLE → blocked

```bash
$ dbguard review-sql "DROP TABLE users;"
```

**Expected output:**

```
Decision: BLOCK
Severity: CRITICAL

Statement 1:
  Type: DROP_TABLE
  Table: users
  Triggered Rules:
    - drop_table
  Reason:
    - DROP TABLE statement
  ...
```

**Exit code:** `3`

---

### 3. Safe SELECT → allowed

```bash
$ dbguard review-sql "SELECT 1;" --format json
```

**Expected output (excerpt):**

```json
{
  "decision": "allow",
  "severity": "low",
  "statements": [
    {
      "index": 1,
      "type": "SELECT",
      "table": "",
      "findings": []
    }
  ]
}
```

**Exit code:** `0`

---

### 4. Protected table → warn (with policy)

```bash
$ dbguard review-file ./examples/protected_table_update.sql --policy ./examples/dbguard.yaml
```

Example file content: `UPDATE users SET role = 'viewer' WHERE id = 1;`  
With `users` in `protected_tables`, the write is flagged.

**Expected output (excerpt):**

```
Decision: WARN
Severity: MEDIUM

Statement 1:
  Type: UPDATE
  Table: users
  Triggered Rules:
    - writes_to_protected_tables
  Reason:
    - Write to protected table: users
  ...
```

**Exit code:** `2`

---

## Suggested screenshots (capture later)

Screenshots are not generated in this repo. Capture these locally for docs or the README:

| # | Description | Command to run |
|---|-------------|----------------|
| 1 | Terminal: DELETE without WHERE blocked | `dbguard review-sql "DELETE FROM users;"` |
| 2 | Terminal: Safe SELECT allowed with JSON output | `dbguard review-sql "SELECT 1;" --format json` |
| 3 | Terminal: Protected table warning with policy | `dbguard review-file ./examples/protected_table_update.sql --policy ./examples/dbguard.yaml` |

Store under `docs/assets/` (e.g. `demo-block.png`, `demo-allow-json.png`, `demo-protected-warn.png`) and link from the README when added.

---

## Quick reference

| Command | Purpose |
|--------|---------|
| `dbguard review-sql "<sql>"` | Review inline SQL |
| `dbguard review-file <path>` | Review a SQL file |
| `dbguard review-file <path> --policy <yaml>` | Use a policy file |
| `dbguard review-sql "..." --format json` | JSON output for CI |
| `dbguard version` | Print version |

---

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

---

## Exit codes

| Code | Meaning |
|------|--------|
| 0 | Allow — no violations |
| 1 | Internal or tool error |
| 2 | Warn — at least one warning |
| 3 | Block — at least one blocking violation |

**CI:** Fail the job on block (exit 3); optionally fail on warn (exit 2). Example:  
`dbguard review-file migration.sql; exit $?` (fail on any non-zero).

---

## Limitations (v1)

- **PostgreSQL only.** No other dialects.
- **No semantic predicate analysis.** e.g. `DELETE FROM users WHERE 1=1` has a WHERE clause and does not trigger `delete_without_where`.
- **Build:** Requires CGO; first compile can take several minutes (PostgreSQL parser).

See [docs/SPEC.md](docs/SPEC.md) for full specification and limitations.

---

## Local development

```bash
go build ./cmd/dbguard
go test ./...
go fmt ./...
go vet ./...
```

---

## CI

This repo’s CI builds and tests on push/PR to `main` with CGO. See [.github/workflows/ci.yml](.github/workflows/ci.yml).

To **use dbguard in your own CI**, see the example workflow in the repo docs.

---

## Documentation

- [docs/SPEC.md](docs/SPEC.md) — Problem, scope, rules, CLI contract, policy schema
- [docs/MILESTONES.md](docs/MILESTONES.md) — Progress and substeps
- [docs/GUARDRAILS.md](docs/GUARDRAILS.md) — Engineering guardrails
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — Code structure and data flow

---

## License

See repository license.
