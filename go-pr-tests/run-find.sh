#!/bin/bash
set -euo pipefail
[ -n "${DEBUG:-}" ] && set -x
: GITHUB_OUTPUT: "${GITHUB_OUTPUT:=/dev/null}"
: RUNNER_TEMP: "${RUNNER_TEMP:=${TMPDIR:-/tmp}}"

merge_coverage() {
  mergedir=$(mktemp -d -p "${RUNNER_TEMP}" coverage.XXXX)
  go tool covdata textfmt -i "$GOCOVERDIR" -o "${mergedir}/cover.out"
  printf 'coverdir=%s\n' "$mergedir" >> "$GITHUB_OUTPUT"
}
if [[ -n "${INPUT_COVERAGE:-}" ]]; then
  GOCOVERDIR=$(mktemp -d -p "${RUNNER_TEMP}" coverage.XXXX)
  trap 'merge_coverage' EXIT
fi

find .\
    \( -type d -name vendor -o -name .git -o -name _\* \) -prune\
  -o\
    \( -type f -name go.mod \)\
    -printf '%h\0' |
  while read -r -d '' dir; do (
    cd "$dir"
    printf '::group::%s\n' "$(go list -m)"
    trap 'echo ::endgroup::' EXIT
    # shellcheck disable=SC2086
    go test\
      ${INPUT_RACE:+-race}\
      ${INPUT_COVERAGE:+-covermode=atomic}\
      ${INPUT_BUILD_TAGS:+-tags=}${INPUT_BUILD_TAGS:-}\
      ./...\
      ${INPUT_COVERAGE:+-cover -args "-test.gocoverdir=${GOCOVERDIR}"}\
); done
