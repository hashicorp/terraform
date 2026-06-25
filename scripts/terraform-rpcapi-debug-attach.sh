#!/usr/bin/env bash
#
# This script is related to some debugging work for stacks in tfc-agent, see branch - av/tfc-agent-dlv
set -euo pipefail

CONTAINER="${1:-$(docker ps -qf "ancestor=hashicorp/tfc-agent-debug:latest")}"
PID=$(docker exec "$CONTAINER" pgrep -x terraform | head -n1)

if [ -z "$PID" ]; then
  echo "Error: terraform rpcapi process not found in container $CONTAINER" >&2
  exit 1
fi

echo "Attaching Delve to terraform rpcapi (PID $PID) in container $CONTAINER..."

# This attaches to the terraform rpcapi sub-process and then serves a debug server for VSCode to attach to
docker exec --user root "$CONTAINER" dlv attach "$PID" /home/tfc-agent/bin/terraform \
  --headless \
  --listen=:2346 \
  --api-version=2 \
  --accept-multiclient \
  --continue
