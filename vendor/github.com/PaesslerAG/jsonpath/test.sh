#!/bin/bash

# Script that runs tests, code coverage, and benchmarks all at once.

JSONPath_PATH=$HOME/gopath/src/github.com/PaesslerAG/jsonpath

# run the actual tests.
cd "${JSONPath_PATH}"
go test -bench=. -benchmem -coverprofile coverage.out
status=$?

if [ "${status}" != 0 ];
then
	exit $status
fi
