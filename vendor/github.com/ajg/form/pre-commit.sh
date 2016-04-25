#!/bin/bash -eu

# TODO: Only colorize messages given a suitable terminal.
# FIXME: Handle case in which no stash entry is created due to no changes.

printf "\e[30m=== PRE-COMMIT STARTING ===\e[m\n"
git stash save --quiet --keep-index --include-untracked

if go build -v ./... && go test -v -cover ./... && go vet ./... && golint . && travis-lint; then
  result=$?
  printf "\e[32m=== PRE-COMMIT SUCCEEDED ===\e[m\n"
else
  result=$?
  printf "\e[31m=== PRE-COMMIT FAILED ===\e[m\n"
fi

git stash pop --quiet
exit $result
