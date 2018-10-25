#!/bin/bash

# We do not run protoc under go:generate because we want to ensure that all
# dependencies of go:generate are "go get"-able for general dev environment
# usability. To compile all protobuf files in this repository, run
# "make protobuf" at the top-level.

set -eu

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

cd "$DIR"

protoc --go_out=paths=source_relative:. planfile.proto
