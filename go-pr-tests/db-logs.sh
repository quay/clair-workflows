#!/bin/bash
set -euo pipefail
exec 1> "${GITHUB_SUMMARY:-/dev/stdout}"
cat <<.
# Database Logs

Version: $(sudo -u postgres psql -c 'SELECT version();')

<details>

.
echo '```'
sudo journalctl --unit postgresql.service --boot -0
printf '```\n\n</details>\n'
