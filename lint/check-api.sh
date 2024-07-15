#!/bin/bash
[ -n "$DEBUG" ] && set -x
npx widdershins \
  --search false \
  --language_tabs 'python:Python' 'go:Golang' 'javascript:Javascript' \
  --summary \
  ./openapi.yaml \
  -o ./Documentation/reference/api.md
git diff --exit-code
