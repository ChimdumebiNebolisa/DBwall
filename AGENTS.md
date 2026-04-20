# AGENTS.md

## Canonical Commands

- Build: `go build -o dbguard ./cmd/dbguard`
- Test: `go test ./...`
- Lint: `go vet ./...`
- Benchmark: `go run ./benchmark/cmd/dbwallbench --repo-root . --manifest ./benchmark/manifest.json --json-out ./benchmark/results/benchmark_results.json --report-out ./benchmark/reports/benchmark_report.md`

## Benchmark Layout

- Benchmark fixtures live under `benchmark/cases/`
- Benchmark policies live under `benchmark/policies/`
- Benchmark manifest lives at `benchmark/manifest.json`
- Benchmark runner code lives under `benchmark/`
- Raw benchmark outputs live under `benchmark/results/`
- Human-readable benchmark reports live under `benchmark/reports/`

## Benchmark Rules

- Every metric claim must come from saved output artifacts, not estimates or recollection.
- Benchmark runs must be deterministic and reproducible: fixed manifest, fixed case ordering, and sequential execution.
- Benchmark summaries must distinguish measured results from assumptions. If a number was not measured in the saved artifacts, do not present it as a result.
- Do not invent benchmark outcomes, runtime numbers, or quality metrics.
- If the benchmark harness cannot run because required repo structure is missing, report the missing structure explicitly.
- Final changes must leave the repo in a clean, runnable state with the benchmark command usable from the repo root.
