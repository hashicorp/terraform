#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

# This script checks that all files have the appropriate copyright headers,
# according to their nearest .copywrite.hcl config file. The copyright tool
# does not natively support repos with multiple licenses, so we have to
# script this ourselves.

set -euo pipefail

# Find all directories containing a .copywrite.hcl config file
directories=$(find . -type f -name '.copywrite.hcl' -execdir pwd \;)
args=${1:-}

for dir in $directories; do
    cd $dir && pwd && go run github.com/hashicorp/copywrite headers $args
done
