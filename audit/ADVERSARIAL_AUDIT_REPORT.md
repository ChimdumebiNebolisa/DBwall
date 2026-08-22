# DBwall Adversarial Security Audit Report

**Audit status:** AUDIT COMPLETE WITH LIMITATIONS
(Limitation: GitHub-hosted CI/release workflows were inspected statically only; no Linux/macOS runtime was available locally. All other checks ran locally on Windows amd64 in both CGO modes.)

## 1. Repository State

| Item | Value |
|---|---|
| Repository | ChimdumebiNebolisa/DBwall |
| Local path | C:\Users\Chimdumebi\DBwall |
| Branch / commit at audit start | main @ `7294355c51bde6b19c27e2e28cb98255a3a1bb8c` (tag v0.2.0 one commit behind HEAD) |
| Working tree | clean at start |
| Audit date | 2026-08-21 |
| Toolchain | go1.26.1 windows/amd64, git-lfs 3.7.0 |
| C toolchain | gcc not on PATH; msys64 mingw64 gcc 16.1.0 found and used via `CC`+`PATH` |
| Commands permitted | local build/test/vet/benchmark/probes; no remote changes |
| Commands blocked | none after locating msys64 gcc (initially CGO was blocked) |

Note: the task prompt states `ADVERSARIAL_CODEBASE_AUDIT_PROTOCOL.md` was added to the repository root; the file is **not present** in the clone at commit `7294355`. The protocol text was supplied inline in the task message and followed instead. Anomaly recorded.

## 2. Executive Verdict

**Ship only with stated conditions.** Overall codebase risk at audit time: **High** (in specific configurations), **Medium** generally. Audit confidence: High for parser/rules/CLI/policy behavior (all reproduced locally in both modes); Medium for CI/release (static inspection only).

DBwall's deterministic design, dual-mode honesty (`coverage_mode` metadata), strictest-wins aggregation, and fail-closed parse errors are genuine strengths. However, several verified defects invert or silence decisions exactly where the tool claims authority: revocations are punished as grants (full mode), an unbounded `UPDATE` passes silently (core mode), a `UNION` arm's `LIMIT` masks an unbounded read of a protected table (full mode), multi-object `GRANT`s to `PUBLIC` pass with zero findings (core mode), and non-ASCII identifiers are silently mangled (core mode). A perfect saved benchmark (29/29) coexisted with all of these because the corpus never exercised them — confirming that benchmark perfection must trigger scrutiny, not reassurance.

### Top five risks (pre-remediation)

1. F-001 REVOKE→GRANT inversion blocks legitimate privilege revocations (full mode).
2. F-003 Core-mode unbounded-UPDATE bypass via subquery `WHERE`.
3. F-004 Core-mode multi-object `GRANT ... TO PUBLIC` silently allows (zero findings) even when a later object is protected.
4. F-002 Full-mode set-op LIMIT attribution masks unbounded reads of protected tables.
5. F-007/F-006 Core tokenizer corrupts non-ASCII identifiers; BOM files fail entirely — silent protection mismatch vs hard failure.

**Strongest counterargument:** every core-vs-full divergence that matters is disclosed in the README, and `semantic_analysis_incomplete` exists precisely to avoid over-claiming.
**Assessment:** disclosure does not hold for F-001 (wrong-direction decision, not reduced coverage), F-003/F-008 (wrong booleans, not missing features), F-006/F-007 (fail/misparse independent of mode choice), or F-005 (silent allow contradicting full-mode warn for the same statement).

## 3. Build and Verification Results (baseline)

| Check | Command | Result | Notes |
|---|---|---|---|
| Build core | `CGO_ENABLED=0 go build ./...` | Pass | |
| Build full | `CGO_ENABLED=1 go build ./...` | Pass after using msys64 mingw64 gcc (`CC`, `PATH`) | plain PATH lacks gcc → cgo fails |
| Tests core | `CGO_ENABLED=0 go test ./...` | Pass (all packages) | |
| Tests full | `CGO_ENABLED=1 go test ./...` | Pass (all packages) | |
| Vet | `go vet ./...` | Pass both modes | |
| Benchmark harness test | included in tests | Pass | |
| Randomized robustness | 20k mutated inputs × 2 modes | No panics, no hangs | temporary probe |

## 4. Findings Summary

| ID | Severity | Priority | Confidence | Category | Title | Status |
|---|---|---|---|---|---|---|
| F-001 | Critical | P1 | High | Correctness/Security | REVOKE analyzed as GRANT; revocations blocked as expansions | Verified |
| F-002 | High | P1 | High | Security | Set-op arm LIMIT marks whole SELECT bounded (full) | Verified |
| F-003 | High | P1 | High | Security | Any-token WHERE search lets subquery satisfy mutation bounding (core) | Verified |
| F-004 | High | P1 | High | Security | Multi-object GRANT/DROP/TRUNCATE: only first object parsed (core); GRANT to PUBLIC silently allowed | Verified |
| F-005 | Medium | P1 | High | Coverage signaling | Core silently ALLOWS unsupported ALTER/DROP targets where full warns | Verified |
| F-006 | Medium | P1 | High | Correctness | UTF-8 BOM makes any file unparseable in both modes (exit 1) | Verified |
| F-007 | Medium | P2 | High | Security | Byte-wise tokenizer corrupts non-ASCII identifiers (core) → protected-name mismatch | Verified |
| F-008 | Medium | P2 | High | Security | Subquery LIMIT satisfies top-level HasLimit in core (false negative) | Verified |
| F-009 | Low | P2 | High | Usability/CLI | Absolute `--policy` inside CWD rejected due to Rel-based containment check (8.3/case forms) | Verified |
| F-010 | Low | P2 | High | False positive | `FETCH FIRST n ROWS ONLY` not treated as limit in core | Verified |
| F-011 | Low | P2 | High | CLI contract | Invalid `--format` silently falls back to human output | Verified |
| F-012 | Low | P3 | High | Usability | Transaction wrappers/unsupported verbs are hard errors in core (fail-closed) | Verified |
| F-013 | Low | P3 | High | Usability | Comment-only SQL file exits 1 in both modes (fail-closed) | Verified |
| F-014 | Medium | P2 | High | Scope gap | `INSERT INTO t SELECT * FROM protected` records the read but no rule consumes it (both modes) | Verified |
| F-015 | Informational | P3 | High | Documentation | DO/FUNCTION/SET ROLE/etc. are warn-only "unsupported"; escalation never applies to OTHER | Verified |
| F-016 | Informational | P3 | High | Supply chain/CI | Action runs dbguard 3×; checksum grep uses unescaped regex; actions pinned by tag not SHA | Verified |
| F-017 | Informational | P3 | High | Documentation | BASELINE_REPORT.md describes obsolete v0.1 architecture; RELEASE_NOTES has encoding artifacts | Verified |
| F-018 | Low | P3 | High | Correctness | `GRANT ... ON DATABASE x` fabricates bogus relation `database` in core | Verified |
| F-019 | Informational | P3 | High | Policy semantics | Leaf/case-folding matching is intentionally over-broad (safe direction) | Verified |

Detailed evidence for each finding is in §6. Reproduction inputs and observed outputs were captured mechanically via temporary probe tests (removed before remediation commits; raw outputs retained under `/tmp` during the audit session).

## 5. Contradictions and Unverifiable Claims

| Source A | Source B | Contradiction | Authority |
|---|---|---|---|
| README "multi-object GRANT ... only accurate in full mode" | Observed core behavior | Accurate, but README understates impact: result is *allow with zero findings*, not a degraded-but-present signal | Runtime (verified) |
| README "constant-folded trivial predicates beyond TRUE and 1=1" full-only | Observed | Accurate (2>1, 'x'='x', NOT FALSE fold only in full) | Runtime |
| README exit-code table | Implementation | Matches (0/1/2/3) | Runtime |
| Saved benchmark 100% across 29 cases | Adversarial probes | Corpus did not cover any of F-001..F-008 shapes | Runtime |
| AGENTS.md benchmark command | Harness | Command works as written | Runtime (core mode; requires_full cases skipped) |

Unverifiable locally: release workflow execution on GitHub runners; smoke-pr-gate against published assets; macOS archives. Static inspection found no defects there beyond F-016 notes.

## 6. Detailed Findings (evidence)

### F-001 REVOKE misclassified as GRANT (Critical, P1, High confidence)
- Evidence: `internal/parser/ast_extract_cgo.go:243-279` — `extractGrant`/`extractGrantRole` never check `node.IsGrants`; PostgreSQL parses `REVOKE` into the same `GrantStmt`/`GrantRoleStmt` nodes with `IsGrants=false`.
- Probe (full): `REVOKE ALL ON TABLE users FROM PUBLIC;` → type GRANT, grant_public=true → rule `grant_to_public_on_protected_objects` **block** (policy protecting `users`). `REVOKE pg_read_all_data FROM analyst;` → `grant_high_risk_role_membership` **block** under every policy tested.
- Impact: security-tightening statements are blocked; teams learn to ignore the gate. Also false confidence: revokes are never actually reviewed as revokes.
- Counterargument: revokes are rare in generated SQL. Assessment: they are common in hardening migrations — exactly the workflows this gate is advertised for.
- Remediation: branch on `IsGrants`; introduce `StmtTypeRevoke`; grant rules skip it. Regression tests both modes.

### F-002 UNION-arm LIMIT bounds whole statement (High, P1)
- Evidence: `internal/parser/ast_extract_cgo.go:103-114` — recursion shares one accumulator; any arm's `LimitCount` sets `stmt.HasLimit` for the parent.
- Probe (full): `(SELECT * FROM logs LIMIT 1) UNION ALL (SELECT * FROM users);` with `users` protected → **allow**, expected warn(s). Verified end-to-end.
- Remediation: compute boundedness per leaf arm; parent is bounded iff its own LimitCount exists or ALL arms are bounded.

### F-003 Subquery WHERE satisfies HasWhere in core (High, P1)
- Evidence: `internal/parser/parse_shared.go:230,249,344` — `containsKeyword(tokens,"WHERE")` scans every token including subqueries.
- Probe (core): `UPDATE accounts SET balance = (SELECT 0 WHERE TRUE);` → **allow**; full mode correctly blocks (`update_without_where`). Verified end-to-end.
- Remediation: depth-aware keyword detection (paren-depth 0) for WHERE/LIMIT in the token parser.

### F-004 Multi-object statements keep only first object in core (High, P1)
- Evidence: `internal/parser/parse_shared.go:255-302` (DROP), `:373-385` (TRUNCATE), `:405-432` (GRANT object loop returns after first identifier).
- Probe (core): `GRANT SELECT ON TABLE orders, users TO PUBLIC;` with `users` protected → **allow, zero findings** (file with only this statement). DROP/TRUNCATE lose secondary objects (aggregate decision may still block via type rules; the protected-object signal is lost).
- Counterargument: README documents reduced coverage. Assessment: disclosure exists, but an allow with zero findings is the strongest possible false-confidence signal; the token-level fix is small, so risk justifies implementing list parsing rather than documenting the hole.

### F-005 Unsupported ALTER/DROP targets silently allowed in core (Medium, P1)
- Evidence: `internal/parser/parse_shared.go:202-210` sets reason `unsupported_statement_in_core_mode`; `internal/rules/rules.go:293-304` filters it from `actionableIncompleteReasons`, so no finding is emitted. Full mode emits `semantic_analysis_incomplete` warn for the same input.
- Probes (core): `ALTER ROLE app WITH SUPERUSER;`, `ALTER SYSTEM SET wal_level='minimal';`, `ALTER SEQUENCE s RESTART;`, `DROP TYPE t;` → allow/no findings; full → warn.
- Remediation: make that reason actionable (or replace with a granular reason) so core warns like full.

### F-006 UTF-8 BOM breaks parsing in both modes (Medium, P1)
- Probe: BOM-prefixed `DELETE FROM logs;` → exit 1 in both modes ("syntax error at or near \xef").
- Remediation: strip a leading UTF-8 BOM once in each Parse entry point; add regression tests.

### F-007 Byte-wise tokenizer corrupts non-ASCII identifiers (Medium, P2)
- Evidence: `tokenizeSQL` iterates bytes (`rune(sql[i])` on single bytes), splitting UTF-8 sequences; probe shows `DELETE FROM païment;` parsed with a mangled table name (silent mismatch vs policy entries) while `"païment"` works.
- Remediation: decode runes (`utf8.DecodeRuneInString`) throughout tokenizer/splitter; property test with non-ASCII identifiers.

### F-008 Subquery LIMIT satisfies top-level HasLimit in core (Medium, P2)
- Probe (core): `SELECT * FROM users WHERE id IN (SELECT id FROM allowed_ids LIMIT 100);` → missing `select_without_limit_from_protected_table` (fires in full). Same fix as F-003 (depth-0 scan).

### F-009 Absolute policy paths inside CWD rejected on form mismatch (Low, P2)
- Evidence: `internal/policy/load.go:13-43` compares `filepath.Rel(cwd, abs)`; Windows 8.3 short names (`CHIMDU~1`) vs long names defeat containment recognition. Probe: running with CWD = temp dir and absolute path to a policy inside it → "outside the working directory".
- Remediation: normalize both sides (EvalSymlinks/case-insensitive compare) before rejecting; keep rejecting true escapes. Relative-path behavior unchanged.

### F-010..F-013 (Low) — see summary table
- F-010: core misses `FETCH FIRST n ROWS ONLY` as a bound → extra warn (FP direction). Fixed alongside F-003 by recognizing `FETCH ... ROWS ONLY` at depth 0.
- F-011: `--format xml` prints human output, exit 0. Fix: validate format, error exit 1.
- F-012/F-013: fail-closed behaviors (BEGIN/COMMIT, unsupported verbs, comment-only files) — accepted as safe-direction; documented, not changed.

### F-014 INSERT..SELECT read source unconsumed (Medium, P2, scope gap)
- Probe: `INSERT INTO archive SELECT * FROM users;` with `users` protected → allow both modes although relations record `read:users`. Declared scope includes "suspicious bulk-access patterns" from protected objects; a staging-table copy evades every bulk_access rule. Resolution chosen: extend `select_without_limit_from_protected_table` semantics? No — adding a new rule mid-release changes product surface; instead this is **documented as an explicit exclusion** in README ("What DBwall Does Not Do") and tracked as a future rule candidate. Rationale: prompt instructs adding rules only when justified; the honest near-term fix is precise documentation of the boundary plus corpus cases pinning current behavior.

### F-015..F-019 — accepted/documented items (see table)
Includes: warn-only treatment of unsupported statement families (F-015), action triple-execution/checksum-regex/unpinned-actions (F-016), stale docs (F-017), `ON DATABASE` bogus relation (F-018, fixed with F-004 work), intentional over-broad protected matching (F-019).

## 7. Rule-by-Mode Capability Matrix

See `audit/CORE_FULL_CAPABILITY_MATRIX.md` (post-fix state described there reflects remediation; pre-fix divergences noted inline).

## 8. Rejected Hypotheses

1. **Dollar-quote splitting breaks nested/same-tag bodies** — rejected: first-close semantics match PostgreSQL; verified via `$tag$...$tag$` and mixed-tag probes.
2. **Findings ordering is nondeterministic** — rejected: `dedupeFindings` uses stable sort by (rule,message); repeated runs produced identical JSON/SARIF.
3. **Empty SARIF runs emit invalid `null`** — rejected: code explicitly initializes `[]sarifResult{}`; validated by parse.
4. **Benchmark metric math double-counts warn/block** — rejected: FP/FN defined solely against `block` positive class; accuracy is exact-match; definitions embedded in artifacts.
5. **Duplicate YAML keys silently weaken policy** — rejected: yaml.v3 rejects duplicates (exit 1 verified).
6. **Policy path traversal escapes CWD** — rejected: `..` rejected (verified); symlinked escape remains theoretical (noted).
7. **Parser panics on hostile input** — rejected at 20k random inputs × both modes; fuzz targets added post-audit.
8. **`containsStarProjection` counts `count(*)` as star projection in full mode** — rejected: AST-based `selectListHasStar` inspects ColumnRef fields only (core still flags `count(*)` — conservative FP, kept).
9. **REVOKE also mishandled in core** — rejected: core fails closed (exit 1) rather than deciding wrongly; fixed anyway for usability parity.

## 9. Remediation Plan

P1 (this change set): F-001, F-002, F-003, F-004 (+F-018), F-005, F-006.
P2 (this change set): F-007, F-008 (same mechanism as F-003), F-009, F-010, F-011; F-014 documented exclusion + corpus pins.
P3: F-012/F-013 documented; F-016 recommendations left to maintainer (SHA pinning cannot be verified offline here); F-017 historical-doc annotations.

## 10. Residual Risk and Unknowns

- pg_query_go v5 C library behavior beyond Go API surface (upstream supply chain) — not auditable here.
- Release binaries on Linux/macOS runners; smoke workflow against live release assets — static review only.
- Symlink-following policy loads inside CWD could reference outside content (defense-in-depth gap; low likelihood).
- Semantic predicate analysis remains syntactic constant folding; column-vs-column tautologies (`id = id`) are intentionally non-trivial (documented limitation).
- `INSERT INTO .. SELECT` bulk-read staging path remains outside rule surface (documented exclusion, F-014).

## 11. Remediation Outcome (recorded after implementation)

All P1 and selected P2 findings were fixed in this change set, each with regression tests. Verification: `CGO_ENABLED=0 go test ./...` and `CGO_ENABLED=1 go test ./...` both pass; `go vet ./...` clean in both modes; `FuzzParse` ran 45s per mode (~1.23M core execs, ~391k full execs) with zero crashes.

| Finding | Resolution | Regression tests |
|---|---|---|
| F-001 REVOKE as GRANT | `IsGrant` honored; new `StmtTypeRevoke`; grant rules skip revokes; REVOKE parses in core instead of exit 1 | `rules_revoke_test.go`, `rules_revoke_full_test.go` (cgo), `TestParse_RevokeIsRevoke`, e2e revoke case |
| F-002 set-op LIMIT leak | per-arm boundedness via `selectArmBounded`; parent bounded iff own limit or all arms bounded | `TestFullMode_UnionArmLimitDoesNotBoundOtherArm`, `TestFullMode_AllArmsBoundedMeansBounded` |
| F-003/F-008 subquery WHERE/LIMIT leakage in core | depth-aware keyword scanning (`indexKeywordTopLevel`) for WHERE/SET/LIMIT/FETCH | `TestParse_SubqueryWhereDoesNotBoundUpdate`, `TestParse_SubqueryLimitDoesNotBoundOuterSelect`, e2e subquery case |
| F-004/F-018 multi-object + ON DATABASE | comma-list parsing for DROP TABLE/TRUNCATE/GRANT objects incl. SCHEMA lists; DATABASE grants recorded without bogus relations (full mode now too) | `TestParse_MultiObject*`, `TestCoreMode_MultiObjectGrantBlocksProtectedSecondTarget`, `TestParse_GrantOnDatabaseDoesNotFabricateRelation`, benchmark cases |
| F-005 silent core allow of unsupported shapes | `unsupported_statement_in_core_mode` is actionable again; core warns like full mode | `TestCoreMode_UnsupportedAlterTargetWarns`, `TestCheck_UnsupportedCoreStatementFiresIncomplete`, benchmark cases |
| F-006 UTF-8 BOM | stripped in both Parse entry points | `TestParse_UTF8BOMStripped`, e2e BOM case, benchmark case |
| F-007 non-ASCII identifiers | rune-aware tokenizer and dollar-tag scanner | `TestParse_NonASCIIIdentifierPreserved`, property test with non-ASCII seeds, benchmark case |
| F-008/F-010 FETCH FIRST | recognized as bound at top level in core | `TestParse_FetchFirstCountsAsLimit`, benchmark case |
| F-009 absolute policy path false rejection | containment check normalizes via `GetLongPathNameW` (Windows) / `EvalSymlinks` (other), case-insensitive compare; true traversal still rejected (verified) | existing `TestLoadFromFile_PathTraversal` still green; manual verification recorded in audit session log |
| F-011 invalid --format | validated in all three commands; error exit 1 | e2e invalid-format case |
| F-013 comment-only phantom statements | splitter emits segments only when real SQL content exists; comment-only files are zero-statement allows | `TestParse_CommentOnlyInputYieldsNoStatements`, `TestParse_TrailingCommentAfterStatement`, benchmark case |
| F-014 INSERT..SELECT read gap | documented exclusion (README) + corpus pinning; rule surface unchanged by design | README "Known exclusions" |

Benchmark artifacts regenerated from scratch after fixes: full mode 41/41 exact (17 block / 9 allow / 15 warn), core mode 26/26 exact on its subset; zero FP/FN in both. Artifacts: `benchmark/results/benchmark_results{,_core}.json`, `benchmark/reports/benchmark_report{,_core}.md`. README numbers updated exclusively from these artifacts.

Remaining open items (documented, not blocking): F-012 transaction-wrapper hard errors in core (fail-closed), F-015 warn-only treatment of unsupported families (design boundary), F-016 CI supply-chain hardening recommendations (SHA-pinning actions requires online verification), F-014 rule-surface decision deferred to maintainers, symlink-based policy escape (defense-in-depth).
