#!/bin/bash
[ -n "$DEBUG" ] && set -x
# Check for all the characters Windows hates.
while read -r file; do
  printf '::error file=%s,title=Bad Filename::Disallowed character in file name.\n' "$file"
  : $(( ct++ ))
done < <(git ls-files -- ':/:*[<>:"|?*]*')
exit "${ct-0}"
