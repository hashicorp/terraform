#!/usr/bin/env bash

# Check goimports
echo "==> Checking the code complies with goimports requirements..."
target_files=$(git diff --name-only origin/main --diff-filter=MA | grep "\.go")

if [[ -z ${target_files}  ]]; then
  exit 0
fi

goimports_files=$(goimports -w -l "${target_files}")
if [[ -n ${goimports_files} ]]; then
  echo 'goimports needs running on the following files:'
  echo "${goimports_files}"
  echo "You can use the command and flags \`goimports -w -l\` to reformat the code"
  exit 1
fi

exit 0
