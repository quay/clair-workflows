#!/bin/bash
[ -n "${DEBUG:-}" ] && set -x
shopt -s lastpipe
: "${GITHUB_STEP_SUMMARY:=/dev/stderr}"
: "${GITHUB_OUTPUT:=/dev/null}"

go list -f '{{if eq "main" .Name}} - {{.ImportPath}}{{end}}' ./... |
  grep cmd | grep -v internal |
  read -r cmds

{
if [[ "${#cmds}" -ne 0 ]]; then
  echo '##' Found commands:
  echo
  echo "${cmds}"
  echo
else
  echo '##' No commands.
fi
} >"$GITHUB_STEP_SUMMARY"
printf 'has-cmds=%d\n' "$(echo "$cmds" | wc -l)" >> "$GITHUB_OUTPUT"
