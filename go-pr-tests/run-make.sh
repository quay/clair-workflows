#!/bin/bash
set -euo pipefail
[ -n "${DEBUG:-}" ] && set -x
: MAKE_TARGET: "${INPUT_MAKE_TARGET?missing 'INPUT_MAKE_TARGET' environemnt variable}"
exec make "$INPUT_MAKE_TARGET"
