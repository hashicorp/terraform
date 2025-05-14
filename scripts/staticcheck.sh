#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1


echo "==> Checking that code complies with static analysis requirements..."
# Skip legacy code which is frozen, and can be removed once we can refactor the
# remote backends to no longer require it.
skip="internal/legacy|backend/remote-state/"

# Skip generated code for protobufs.
skip=$skip"|internal/planproto|internal/tfplugin5|internal/tfplugin6"

packages=$(go list ./... | egrep -v ${skip})

# Note that we globally disable some checks. The list is controlled by the
# top-level staticcheck.conf file in this repo.
go tool honnef.co/go/tools/cmd/staticcheck ${packages}
