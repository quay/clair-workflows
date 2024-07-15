#!/bin/bash
[ -n "$DEBUG" ] && set -x
# Make sure every module is tidy.
find . -not \( -name '.?*' -prune \) -name go.mod\
  \(\
  -execdir sh -ec 'go mod tidy; git diff --exit-code' \;\
  -o\
  -printf '::error file=%p,title=Tidy Check::Commit would leave go.mod untidy\n'\
  \)
