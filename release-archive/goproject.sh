#!/bin/bash
[ -n "${DEBUG:-}" ] && set -x
: "${VERSION:=$(git describe --match 'v*' --long | sed 's/\(.\+\)-\([0-9]\+-g[a-f0-9]\)/\1+\2/')}"
: "${PROJECT_NAME:=$(basename "$(pwd)")}"
: "${GITHUB_STEP_SUMMARY:=/dev/stderr}"
tarball="${PROJECT_NAME}-${VERSION}.tar"
prefix="${PROJECT_NAME}-${VERSION}/"

printf '## Creating Archive\n\n'
printf 'making archive for version: %s\n\n' "${VERSION}"
git archive --format tar\
  --prefix "$prefix"\
  --output "$tarball"\
  "$VERSION"

{
cat << '.'
### `go mod vendor` output:

```
.
go mod vendor ||:
if [[ -d vendor ]]; then
  dt=$(tar --list --file "$tarball" --utc --full-time "${prefix}go.mod" | awk '{print $4 "T" $5 "Z"}')
  tar --append --file "$tarball"\
    --transform "s,^,${prefix},"\
    --mtime "$(date -Iseconds --date "$dt")"\
    --sort name\
    --pax-option=exthdr.name=%d/PaxHeaders/%f,delete=atime,delete=ctime\
    vendor
fi
cat << '.'
```

### `zstd` output:

```
.
zstd --rm -f "$tarball"
echo '```'
} >"$GITHUB_STEP_SUMMARY"
