#!/bin/bash
[ -n "$DEBUG" ] && set -x
: base ref: "\"${GITHUB_BASE_REF-}\""
from="origin/${GITHUB_BASE_REF:-main}"
fmt='::warning title=Spelling Error,file={{.Filename}},line={{.Line}},col={{.Column}}::{{printf "%q" .Original}} is a misspelling of {{printf "%q" .Corrected}}'
# These don't error... yet.
git diff -z --name-only "${from}..." -- '*.go' |
  xargs -0 go run github.com/golangci/misspell/cmd/misspell@latest -f "$fmt" -source go
git diff -z --name-only "${from}..." -- '*.md' |
  xargs -0 go run github.com/golangci/misspell/cmd/misspell@latest -f "$fmt" -source text
