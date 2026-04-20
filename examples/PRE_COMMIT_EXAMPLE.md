# Pre-commit Example

Use DBwall in a local Git hook so dangerous PostgreSQL changes are caught before commit.

## Simple shell hook

```bash
#!/usr/bin/env bash
set -euo pipefail

changed_sql=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.sql$' || true)
if [ -z "${changed_sql}" ]; then
  exit 0
fi

for file in ${changed_sql}; do
  ./dbguard review-file "${file}" --policy ./dbguard.yaml
done
```

Save that as `.git/hooks/pre-commit` and make it executable.

## pre-commit framework snippet

```yaml
repos:
  - repo: local
    hooks:
      - id: dbwall
        name: dbwall
        entry: ./dbguard review-file
        language: system
        files: \.sql$
        pass_filenames: true
        args: ["--policy", "./dbguard.yaml"]
```
