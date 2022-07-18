#!/usr/bin/env bash

# Check go vet
echo "==> Checking that the code complies with go vet requirements..."
govet_out=$(go vet ./... 2>&1)
if [[ -n ${govet_out} ]]; then
  echo "go vet has discovered the following issues"
  echo "${govet_out}"
  exit 1
fi

exit 0
