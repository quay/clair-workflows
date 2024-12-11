#!/bin/bash
set -euo pipefail
[ -n "${DEBUG:-}" ] && set -x
: GITHUB_SUMMARY: "${GITHUB_SUMMARY:=/dev/null}"
{
  cat <<'.'
## go environment:

<details>

```
.
  go env
  cat <<'.'

```

</details>
.
} >> "${GITHUB_SUMMARY}"
