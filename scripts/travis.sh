#!/bin/bash

set -e
echo "" > coverage.txt

for d in $(go list ./... | grep -v vendor); do
    go test -mod=vendor -timeout=2m -parallel=4 -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
