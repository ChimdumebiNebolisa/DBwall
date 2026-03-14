# dbguard – Baseline Investigation Report

**Date:** (investigation run)  
**Branch:** main  
**Remote:** origin → https://github.com/ChimdumebiNebolisa/DBwall.git  
**Working tree:** clean

---

## 1. Repo inspection

- **Layout:** cmd/dbguard, internal/{cli,analyzer,parser,policy,report,rules,version}, examples/, .github/workflows/ci.yml, README.md, go.mod. Docs (SPEC, GUARDRAILS, ARCHITECTURE, MILESTONES) exist under docs/ (gitignored).
- **Parser:** internal/parser/parse.go uses pg_query_go v5 `ParseToJSON()`, then unmarshals JSON into a struct expecting `stmts[].stmt` as a map with one key per statement type (e.g. DeleteStmt, UpdateStmt). Relation extracted via `relation` field and `relationRelname()` (handles direct `relname` or nested `RangeVar.relname`). WHERE via `where_clause` or `whereClause`. DROP TABLE uses `remove_type == 1` and `objects`; table name from first element (list of list or RangeVar). ALTER TABLE uses `cmds` and `subtype == 11` or string `AT_DropColumn`.
- **Rules:** internal/rules/rules.go – all five v1 rules implemented; depend on parser.Statement Type, Table, HasWhere.
- **Analyzer:** internal/analyzer/analyzer.go – aggregates findings; strictest decision/severity wins.
- **CLI:** internal/cli/review.go – loadPolicy → Parse → Analyze → report → decisionToExit (block=3, warn=2, allow=0, default allow).

---

## 2. Build and tests

- **go mod tidy / go build / go test:** **NOT RUN** in this environment (Go not in PATH).
- **Smoke checks (1–9):** **NOT RUN** (binary not built).

---

## 3. Smoke check commands and expected behavior (contract)

| # | Command | Expected decision | Expected exit code |
|---|--------|-------------------|---------------------|
| 1 | `./dbguard review-sql "DELETE FROM users;"` | BLOCK | 3 |
| 2 | `./dbguard review-sql "UPDATE users SET role='admin';"` | BLOCK | 3 |
| 3 | `./dbguard review-sql "DROP TABLE users;"` | BLOCK | 3 |
| 4 | `./dbguard review-file ./examples/drop_column.sql` | BLOCK | 3 |
| 5 | `./dbguard review-sql "SELECT 1;" --format json` | valid JSON, ALLOW | 0 |
| 6 | `./dbguard review-sql "UPDATE users SET name='x' WHERE id=1;"` | ALLOW (unless protected) | 0 |
| 7 | `./dbguard review-sql "DELETE FROM users WHERE id=1;"` | ALLOW (unless protected) | 0 |
| 8 | `./dbguard review-sql "DELETE FROM users; SELECT 1;"` | BLOCK | 3 |
| 9 | `./dbguard review-file ./examples/protected_table_update.sql --policy ./examples/dbguard.yaml` | reflect protected-table rule (warn) | 2 |

---

## 4. Risk areas (from code inspection)

1. **Parser JSON shape**
   - pg_query_go JSON may use different key names (e.g. snake_case vs camelCase for `where_clause`). Code checks both.
   - Top-level `stmts[].stmt` as map with one key (e.g. DeleteStmt) matches documented shape.

2. **Relation extraction**
   - Code expects `relation` with either `relname` or `RangeVar.relname`. If the library wraps relation in another oneof key, extraction could fail.

3. **DROP TABLE objects**
   - Code expects `objects` as list; first element either list (qualified name, last element = table) or map (RangeVar/relname). Actual pg_query_go encoding for DROP TABLE objects may differ (e.g. different node types).

4. **ALTER TABLE cmds**
   - Code expects `cmds[]` with `subtype` 11 or string `AT_DropColumn`. Protobuf enum values may differ in JSON.

5. **remove_type**
   - Code uses `remove_type == 1` for OBJECT_TABLE; enum value must match pg_query_go’s encoding.

6. **Multi-statement**
   - Aggregation and exit code logic are correct; failure would come from parser not returning multiple statements.

7. **Protected tables**
   - Depends on parser setting `stmt.Table` for UPDATE/DELETE; if relation extraction fails, protected-table rule will not fire.

---

## 5. Next steps

- Run `go test ./...` and the nine smoke checks in an environment where Go is available (or in CI).
- If any smoke check fails: trace to parser (JSON/extraction), rules, analyzer, or CLI and apply minimal fix.
- Add or extend regression tests that use the real parser for: DELETE without/with WHERE, UPDATE without/with WHERE, DROP TABLE, ALTER TABLE DROP COLUMN, protected table, multi-statement, JSON output, exit codes.
- After each fix: one change unit → test → commit → push.

---

## 6. Baseline result summary

| Item | Status |
|------|--------|
| Repo / branch / remote | OK |
| Code and docs read | Done |
| go build | Not run (Go not in PATH) |
| go test | Not run |
| Smoke checks 1–9 | Not run |
| Pass/fail vs contract | N/A until smoke checks run |

**Conclusion:** Baseline code review and contract are documented. Verification of behavior (build, tests, smoke checks) must be done in an environment with Go available; then fix any mismatches and add regression tests, one change unit at a time with push after each.
