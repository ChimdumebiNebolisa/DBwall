# dbguard – Architecture

## Overview

Layers are separated to keep parser, policy, rules, and CLI independent.

## Layers

1. **CLI** (`internal/cli`) – Argument parsing, commands, input loading, output format selection, exit code handling.

2. **Policy** (`internal/policy`) – Default policy, YAML loading, validation, rule action lookup.

3. **Parser** (`internal/parser`) – PostgreSQL parsing wrapper; translates parser output into analyzer-friendly statement models. Parser-specific code isolated here.

4. **Rules / Analyzer** (`internal/rules`, `internal/analyzer`) – Operate on parsed statements. Each rule independently testable. Findings combined into overall decision.

5. **Report** (`internal/report`) – Human and JSON output with stable formatting.

## Data flow

1. CLI loads SQL (string or file) and optional policy path.
2. Policy layer loads and validates YAML (or uses defaults).
3. Parser turns SQL into statement list / AST representation.
4. Analyzer runs rules over each statement, collects findings.
5. Aggregation produces overall decision (allow/warn/block) and severity.
6. Report layer formats output; CLI sets exit code and prints.

## Package layout

- `cmd/dbguard` – Entrypoint; wires Cobra and delegates to internal packages.
- `internal/cli` – Command definitions and I/O.
- `internal/parser` – Parser wrapper and statement models.
- `internal/policy` – Policy structs and loading.
- `internal/rules` – Individual rule implementations.
- `internal/analyzer` – Orchestration and decision aggregation.
- `internal/report` – Output formatters.
- `internal/version` – Version constant.

Details will be expanded as implementation progresses.
