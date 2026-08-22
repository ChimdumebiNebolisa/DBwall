# DBwall Core vs Full Capability Matrix

Legend: ✅ accurate · ⚠️ partial/conservative (documented direction) · ❌ wrong or silently missing (pre-fix) → fixed in remediation unless noted.
"Post-fix" columns describe behavior after the audit remediation change set (commit history records the transition).

| # | Statement family / capability | Core (CGO=0) pre-fix | Full (CGO=1) pre-fix | Core post-fix | Full post-fix |
|---|---|---|---|---|---|
| 1 | DELETE/UPDATE unbounded + trivial `TRUE`/`1=1` | ✅ | ✅ | ✅ (subquery WHERE no longer counts, F-003) | ✅ |
| 2 | Constant-folded tautologies beyond TRUE/1=1 (`2>1`, `'x'='x'`, `NOT FALSE`, `x OR TRUE`) | ⚠️ treated non-trivial (documented) | ✅ folded | ⚠️ unchanged (documented divergence) | ✅ |
| 3 | UPDATE bounded via subquery-in-SET containing WHERE | ❌ counted as bounded → silent allow (F-003) | ✅ predicate absent detected | ✅ depth-0 scan | ✅ |
| 4 | DELETE ... USING secondary relations | ⚠️ source not captured; predicate still evaluated at depth 0 post-fix | ✅ read relation captured | ⚠️ unchanged (documented) | ✅ |
| 5 | UPDATE ... FROM | ⚠️ same as #4 | ✅ | ⚠️ unchanged (documented) | ✅ |
| 6 | Join secondary relations in reads | ❌ first FROM only | ✅ | ⚠️ unchanged (documented) | ✅ |
| 7 | CTEs incl. nested CTE mutations (`WITH del AS (DELETE ...)`) | ❌ parse error (fail-closed exit 1) | ✅ | ❌→unchanged: still fail-closed (F-012 accepted) | ✅ |
| 8 | Subquery sources `FROM (SELECT ...)` | ❌ relations lost | ✅ | ⚠️ unchanged (documented) | ✅ |
| 9 | Multi-object `DROP TABLE a, b` | ⚠️ first object only (F-004); type rule still blocks | ✅ all objects | ✅ list parsed (protected-write signal restored) | ✅ |
| 10 | Multi-object `TRUNCATE a, b` | ⚠️ first object only (F-004) | ✅ | ✅ list parsed | ✅ |
| 11 | Multi-object `GRANT ... ON TABLE a, b TO PUBLIC` | ❌ first object only → protected second object silently allowed (F-004) | ✅ | ✅ list parsed; allow→correct block | ✅ |
| 12 | `GRANT ... ON SCHEMA s TO PUBLIC` | ✅ (schema branch + fallback) | ✅ | ✅ | ✅ |
| 13 | `GRANT ... ON DATABASE d TO PUBLIC` | ❌ fabricated target relation `database` (F-018) | ⚠️ incomplete reason `grant_partial_object_type` | ✅ object recorded as database name, no bogus relation | ⚠️ unchanged (warn via incompleteness) |
| 14 | Role membership grants (`GRANT r TO u`) incl. high-risk builtins | ✅ | ✅ | ✅ | ✅ |
| 15 | REVOKE (object & role forms) | ❌ parse error (exit 1) | ❌ analyzed as GRANT → false blocks (F-001 Critical) | ✅ recognized, allowed by design (no revoke rules yet) | ✅ IsGrants honored |
| 16 | ALTER DEFAULT PRIVILEGES ... TO PUBLIC | ✅ | ✅ | ✅ | ✅ |
| 17 | ALTER TABLE (DROP COLUMN / constraint / NOT NULL) | ✅ | ✅ | ✅ | ✅ |
| 18 | Other ALTER targets (ROLE/SYSTEM/SEQUENCE/...) | ❌ silent allow (F-005) | ⚠️ warn via unsupported | ✅ warn (actionable reason) | ⚠️ warn (unchanged) |
| 19 | DROP SCHEMA / DROP DATABASE | ✅ | ✅ | ✅ | ✅ |
| 20 | COPY table/query TO STDOUT/PROGRAM from protected source | ✅ (token heuristic finds query source) | ✅ AST-verified | ✅ | ✅ |
| 21 | SELECT limit detection: top-level LIMIT | ✅ | ✅ | ✅ | ✅ |
| 22 | Subquery LIMIT leaking into outer HasLimit | ❌ false negative on protected-read warn (F-008) | n/a (AST scoped) | ✅ depth-0 scan | ✅ |
| 23 | Set-op arms: any arm LIMIT bounds everything | n/a (parse error) | ❌ masks unbounded arm (F-002) | n/a | ✅ per-arm boundedness |
| 24 | `FETCH FIRST n ROWS ONLY` as bound | ❌ extra warn FP (F-010) | ✅ | ✅ | ✅ |
| 25 | Quoted identifiers / case folding | ✅ PG rules (preserve quoted, fold unquoted) | ✅ | ✅ | ✅ |
| 26 | Non-ASCII identifiers | ❌ mangled names (F-007) | ✅ | ✅ rune-aware tokenizer | ✅ |
| 27 | UTF-8 BOM input | ❌ hard parse error (F-006) | ❌ same | ✅ BOM stripped | ✅ BOM stripped |
| 28 | Comments between tokens / multiline / dollar quotes | ✅ | ✅ | ✅ | ✅ |
| 29 | Unsupported families (DO, CREATE FUNCTION/VIEW/TABLE/POLICY/EXTENSION, SET ROLE, SET SESSION AUTHORIZATION, VACUUM, COMMENT, PREPARE/EXECUTE, REFRESH MV) | mixed: some parse-error, ALTER-family silent allow (F-005) | ⚠️ warn `semantic_analysis_incomplete`; never escalates for OTHER (F-015) | ⚠️ consistent warn post-fix | ⚠️ unchanged (design boundary documented) |
| 30 | BEGIN/COMMIT/SAVEPOINT wrappers | ❌ exit 1 (fail-closed, F-012) | ⚠️ unsupported warns per statement | ⚠️ unchanged | ⚠️ unchanged |
| 31 | INSERT .. SELECT read-source protection | ❌ source not captured; no consumer anyway | ⚠️ source captured but no rule consumes it (F-014 documented exclusion) | ⚠️ unchanged (documented exclusion) | ⚠️ unchanged |
| 32 | Per-statement completeness metadata + `coverage_mode` signaling | ✅ JSON/human/SARIF | ✅ | ✅ core reasons now actionable where meaningful (F-005) | ✅ |

## Standing limitations (intentional, documented)

- Predicate analysis is syntactic constant folding; `id = id` and joined-table predicates are not proven safe anywhere.
- No runtime/schema introspection; views are not expanded; RLS is out of scope.
- Core mode remains a reduced parser: joins/subquery sources/CTEs still require full mode (rows 4,6,7,8,31).
- Protected-name matching is deliberately over-broad (leaf match + case folding).
