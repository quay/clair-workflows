#!/bin/bash
[ -n "${DEBUG:-}" ] && set -x
: "${GITHUB_REF?GITHUB_REF not provided}"
git fetch origin "+${GITHUB_REF}:${GITHUB_REF}"
