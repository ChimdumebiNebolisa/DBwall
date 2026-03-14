# dbguard v1.0.0

First stable release of **dbguard**: an AST-based Go CLI that reviews AI-generated PostgreSQL SQL and blocks unsafe operations before execution.

---

## What is dbguard?

dbguard parses PostgreSQL SQL using the same parser as PostgreSQL (via pg_query), applies configurable safety rules, and returns a clear **allow / warn / block** decision. It is built for developers, CI pipelines, and agent toolchains—anywhere AI-generated or ad-hoc SQL might be executed without review.

---

## What v1 supports

- **Input:** SQL as a string (`review-sql`) or from a file (`review-file`).
- **Parser:** PostgreSQL only (pg_query_go); real AST, not regex.
- **Rules (v1):**
  - DELETE without WHERE → block (default)
  - UPDATE without WHERE → block (default)
  - DROP TABLE → block (default)
  - ALTER TABLE … DROP COLUMN → block (default)
  - Writes to protected tables (from policy) → warn (default)
- **Policy:** YAML file for dialect, protected table list, and per-rule action (allow/warn/block).
- **Output:** Human-readable and JSON; exit codes 0 (allow), 1 (error), 2 (warn), 3 (block) for CI.

---

## What v1 does not support

- Other SQL dialects (MySQL, SQLite, etc.)
- Semantic analysis of predicates (e.g. `WHERE 1=1` is treated as having a WHERE)
- Database connections or live schema introspection
- Query plan analysis, row-count estimation, or migration orchestration
- Web UI, cloud service, or LLM integration

---

## Installation and build

- **Requirements:** Go 1.21+ and CGO (C compiler required for pg_query). On Ubuntu, `gcc` is usually present; on Windows, MinGW/WinLibs or similar.
- **Build:**
  ```bash
  git clone https://github.com/ChimdumebiNebolisa/DBwall.git
  cd DBwall
  go mod tidy
  CGO_ENABLED=1 go build -o dbguard ./cmd/dbguard
  ```
- First build can take several minutes (PostgreSQL parser compilation).

---

## Key examples

```bash
# Blocked: DELETE without WHERE
dbguard review-sql "DELETE FROM users;"
# → Decision: BLOCK, exit 3

# Blocked: DROP TABLE
dbguard review-sql "DROP TABLE users;"
# → Decision: BLOCK, exit 3

# Allowed: safe SELECT
dbguard review-sql "SELECT 1;" --format json
# → decision: allow, exit 0

# With policy: protected table write → warn
dbguard review-file ./examples/protected_table_update.sql --policy ./examples/dbguard.yaml
# → Decision: WARN, exit 2
```

---

## Known limitations

- **PostgreSQL only.** No other dialects in v1.
- **No semantic predicate analysis.** e.g. `DELETE FROM t WHERE 1=1` is considered to have a WHERE clause.
- **Build:** CGO required; first compile can be slow.
- **Table names:** Extracted from parse tree; qualified names (schema.table) are normalized to the table part where applicable.

---

## Exit code contract

| Code | Meaning |
|------|--------|
| 0 | Allow — no violations |
| 1 | Internal or tool error |
| 2 | Warn — at least one warning |
| 3 | Block — at least one blocking violation |

Use exit code **3** in CI to fail the job when unsafe SQL is detected.

---

## Who this is for

- Teams that run AI-generated SQL or migrations and want a gate before execution.
- CI pipelines that need to block unsafe changes to migrations or scripts.
- Developers and operators who want a single, local tool for PostgreSQL safety checks without regex or custom parsers.

---

## Repository and docs

- **Repo:** https://github.com/ChimdumebiNebolisa/DBwall
- **README:** [README.md](README.md)
- **Examples:** [examples/](examples/)
- **CI usage:** [examples/GITHUB_ACTION_EXAMPLE.md](examples/GITHUB_ACTION_EXAMPLE.md)

---

## Creating this release manually (when GitHub CLI is available)

If `gh` is installed and authenticated, you can create the GitHub release with:

```bash
git tag -a v1.0.0 -m "dbguard v1.0.0"
git push origin v1.0.0
gh release create v1.0.0 --title "dbguard v1.0.0" --notes-file RELEASE_NOTES_v1.0.0.md
```

To attach a built binary (e.g. for Linux):

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o dbguard_linux_amd64 ./cmd/dbguard
gh release create v1.0.0 --title "dbguard v1.0.0" --notes-file RELEASE_NOTES_v1.0.0.md dbguard_linux_amd64
```
