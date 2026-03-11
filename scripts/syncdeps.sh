#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

# This repository contains one logical codebase but as an implementation detail
# it's split into multiple modules at the boundaries of code ownership, so
# that we can keep track of which dependency changes might affect which
# components.
#
# This script runs "go mod tidy" in each module to synchronize any dependency
# updates that were made in any one of the modules.

set -eufo pipefail

# We'll do our work in the root of the repository, which is the parent
# directory of where this script is.
cd "$( dirname "${BASH_SOURCE[0]}" )/.."

# We need to make sure the root go.mod is synchronized first, because otherwise
# the "go list" command below might fail.
go mod tidy

# Each of the modules starting at our root gets its go.mod and go.sum
# synchronized, so that we can see which components are affected by an
# update and therefore which codeowners might be interested in the change.
for dir in $(go list -m -f '{{.Dir}}' github.com/hashicorp/terraform/...); do
    (cd $dir && go mod tidy)
done
