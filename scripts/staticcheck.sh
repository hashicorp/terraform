#!/usr/bin/env bash

echo "==> Checking that code complies with static analysis requirements..."
# The legacy code is frozen, and should not be updated. It will be removed once
# we can refactor the remote backends to no longer require it.
skip="internal/legacy|backend/remote-state/"
packages=$(go list ./... | egrep -v ${skip})

# We are skipping style-related checks, since terraform intentionally breaks
# some of these. The goal here is to find issues that reduce code clarity, or
# may result in bugs.
staticcheck -checks 'all,-ST*' ${packages}
