#!/usr/bin/env bash
# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: BUSL-1.1


echo "==> Checking for switch statement exhaustiveness..."

# For now we're only checking a handful of packages, rather than defaulting to
# everything with a skip list.
go tool github.com/nishanths/exhaustive/cmd/exhaustive ./internal/command/views/json
