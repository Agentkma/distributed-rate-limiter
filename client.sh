#!/usr/bin/env bash

set -euo pipefail

cat <<'EOF'
Manual test commands:

curl http://localhost:8001/api
curl http://localhost:8002/api
curl http://localhost:8003/api

# 4th request from same client IP (across any server) should return 429:
curl http://localhost:8001/api
EOF