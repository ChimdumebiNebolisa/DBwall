# dbguard – Engineering guardrails

Strict rules for implementation and maintenance.

1. **PostgreSQL only** – v1 supports only PostgreSQL. No dialect abstraction theater.

2. **AST-based analysis only** – Detection must be based on parsed statement structure. No regex-based core detection.

3. **No scope creep** – Do not add features outside the v1 spec. Defer multi-dialect, plan analysis, DB connection, UI, LLM, etc.

4. **Substep discipline** – Every substep ends with tests (where applicable), doc updates, and a git commit + push.

5. **No silent behavior changes** – Document behavior and limitations. Spec and ARCHITECTURE must stay accurate.

6. **Push failures** – If `git push` fails, stop and report the exact error. Do not continue until the push issue is resolved.

7. **No fake completeness** – Do not mark a substep complete unless code, tests, and docs for that substep are done.

8. **Clean commits** – One logical change per commit; message format: `[Mx.y] short description`.

9. **No dead code** – No junk placeholders that are never used. Minimal, purposeful scaffolding.

10. **Honest docs** – README and docs must accurately describe what the tool does and does not do.
