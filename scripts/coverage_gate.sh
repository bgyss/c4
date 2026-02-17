#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-coverage.txt}"

if [[ ! -f "$PROFILE" ]]; then
  echo "coverage gate: profile not found: $PROFILE" >&2
  exit 1
fi

TOTAL_FLOOR="${TOTAL_FLOOR:-95}"
CHANGED_FILE_FLOOR="${CHANGED_FILE_FLOOR:-95}"

critical_files=(
  "db/db.go:97"
  "id/tree.go:97"
  "id/encoding.go:97"
  "manifest/nillist.go:97"
  "manifest/manifest.go:97"
  "store/logger.go:97"
  "store/validating.go:97"
  "cmd/c4/main.go:97"
  "cmd/c4/walker.go:97"
)

coverage_for_suffix() {
  local suffix="$1"
  awk -v sfx="/${suffix}:" '
    NR > 1 && $1 ~ sfx {
      stmts += $2
      if ($3 > 0) covered += $2
    }
    END {
      if (stmts == 0) {
        print "NaN"
        exit 0
      }
      printf "%.2f", 100 * covered / stmts
    }
  ' "$PROFILE"
}

check_floor() {
  local label="$1"
  local got="$2"
  local floor="$3"
  awk -v got="$got" -v floor="$floor" 'BEGIN { exit (got + 0 >= floor + 0 ? 0 : 1) }'
  local ok=$?
  if [[ $ok -ne 0 ]]; then
    echo "coverage gate: FAIL ${label}: ${got}% < ${floor}%"
    return 1
  fi
  echo "coverage gate: PASS ${label}: ${got}% >= ${floor}%"
}

failures=0

total="$(go tool cover -func="$PROFILE" | awk '/^total:/ {gsub("%","",$3); print $3}')"
if ! check_floor "total" "$total" "$TOTAL_FLOOR"; then
  failures=$((failures + 1))
fi

for item in "${critical_files[@]}"; do
  file="${item%%:*}"
  floor="${item##*:}"
  got="$(coverage_for_suffix "$file")"
  if [[ "$got" == "NaN" ]]; then
    echo "coverage gate: FAIL critical ${file}: no coverage data"
    failures=$((failures + 1))
    continue
  fi
  if ! check_floor "critical ${file}" "$got" "$floor"; then
    failures=$((failures + 1))
  fi
done

diff_target=""
if [[ -n "${GITHUB_BASE_REF:-}" ]]; then
  diff_target="origin/${GITHUB_BASE_REF}...HEAD"
elif [[ -n "${GITHUB_EVENT_BEFORE:-}" ]]; then
  diff_target="${GITHUB_EVENT_BEFORE}...HEAD"
elif git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
  diff_target="HEAD~1...HEAD"
fi

changed_files=()
if [[ -n "$diff_target" ]]; then
  while IFS= read -r f; do
    changed_files+=("$f")
  done < <(git diff --name-only "$diff_target" -- '*.go' ':!**/*_test.go')
fi

if [[ ${#changed_files[@]} -gt 0 ]]; then
  echo "coverage gate: checking changed files (${#changed_files[@]}) against ${CHANGED_FILE_FLOOR}% floor"
  for file in "${changed_files[@]}"; do
    got="$(coverage_for_suffix "$file")"
    if [[ "$got" == "NaN" ]]; then
      continue
    fi
    if ! check_floor "changed ${file}" "$got" "$CHANGED_FILE_FLOOR"; then
      failures=$((failures + 1))
    fi
  done
else
  echo "coverage gate: no changed non-test go files detected for changed-file floor"
fi

if [[ $failures -gt 0 ]]; then
  echo "coverage gate: ${failures} check(s) failed"
  exit 1
fi

echo "coverage gate: all checks passed"
