#!/usr/bin/env bash
# Regression checks for action.yml sql-paths parsing rules:
# - newline-delimited
# - spaces inside filenames preserved
# - CRLF stripped safely
# - no splitting on arbitrary whitespace
set -euo pipefail

parse_sql_paths() {
  local input="$1"
  local -n _out="$2"
  _out=()
  while IFS= read -r line || [[ -n "${line}" ]]; do
    line="${line//$'\r'/}"
    [[ -z "${line}" ]] && continue
    _out+=("${line}")
  done < <(printf '%s' "${input}")
}

assert_eq() {
  local want="$1"
  local got="$2"
  local label="$3"
  if [[ "${want}" != "${got}" ]]; then
    echo "FAIL ${label}: want=${want@Q} got=${got@Q}" >&2
    exit 1
  fi
}

paths=()
parse_sql_paths $'migrations/a.sql\nmigrations/my file.sql\n' paths
assert_eq "2" "${#paths[@]}" "count with spaced filename"
assert_eq "migrations/a.sql" "${paths[0]}" "first path"
assert_eq "migrations/my file.sql" "${paths[1]}" "path with spaces"

paths=()
parse_sql_paths $'migrations/a.sql\r\nmigrations/b.sql\r\n' paths
assert_eq "2" "${#paths[@]}" "CRLF count"
assert_eq "migrations/a.sql" "${paths[0]}" "CRLF first"
assert_eq "migrations/b.sql" "${paths[1]}" "CRLF second"

paths=()
parse_sql_paths 'migrations/only one.sql' paths
assert_eq "1" "${#paths[@]}" "single path no trailing newline"
assert_eq "migrations/only one.sql" "${paths[0]}" "single spaced path"

paths=()
parse_sql_paths $'a.sql b.sql\n' paths
assert_eq "1" "${#paths[@]}" "must not split on spaces"
assert_eq "a.sql b.sql" "${paths[0]}" "space-separated stays one path token"

paths=()
parse_sql_paths '' paths
assert_eq "0" "${#paths[@]}" "empty input"

echo "action path parse checks OK"
