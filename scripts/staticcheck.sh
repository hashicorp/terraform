#!/usr/bin/env bash

echo "==> Checking that code complies with static analysis requirements..."
# Skip legacy code which is frozen, and can be removed once we can refactor the
# remote backends to no longer require it.
skip="internal/legacy|backend/remote-state/"

# Skip generated code for protobufs.
skip=$skip"|internal/planproto|internal/tfplugin5|internal/tfplugin6"

packages=$(go list ./... | egrep -v ${skip})

# We are skipping style-related checks, since terraform intentionally breaks
# some of these. The goal here is to find issues that reduce code clarity, or
# may result in bugs.
go run honnef.co/go/tools/cmd/staticcheck -checks 'all,-ST*' ${packages}
