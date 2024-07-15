#!/bin/bash
[ -n "$DEBUG" ] && set -x
# Make sure the Go version is consistent across the repo.
# Operates on all Dockerfile and docker-compose.yaml files in the repo.
version=$(sed -n '/^go /{s/go \(1\.[0-9]\+\)\.[0-9]\+/\1/;p;q}' go.mod)
{
  find . -name Dockerfile -print0 |
    xargs -0 awk -v "want=$version" \
      '/^ARG GO_VERSION/{
        split($2,ver,/=/)
        if(ver[2]!=want) printf "%s\t%d\n", FILENAME, FNR
      }'
  find . -name docker-compose.yaml -print0 |
    xargs -0 awk -v "want=$version" \
     '/&go-image/{
        split($3,ref,/:/)
        if(ref[2]!=want) printf "%s\t%d\n", FILENAME, FNR
      }'
} |
  awk -v "want=$version" \
    '{printf "::error file=%s,line=%d,title=Go Version Skew::Go version does not match `go.mod`: want %s\n", $1, $2, want}'
