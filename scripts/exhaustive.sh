#!/usr/bin/env bash

echo "==> Checking for switch statement exhaustiveness..."

# For now we're only checking a handful of packages, rather than defaulting to
# everything with a skip list.
exhaustive ./internal/command/views/json
