#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

#
# This script builds the application from source for multiple platforms.

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that directory
cd "$DIR"

# Determine the arch/os combos we're building for
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:-linux darwin windows freebsd openbsd solaris}
XC_EXCLUDE_OSARCH="!darwin/arm !darwin/386"

# Delete the old dir
echo "==> Removing old directory..."
rm -f bin/*
rm -rf pkg/*
mkdir -p bin/

# If its dev mode, only build for ourself
if [[ -n "${TF_DEV}" ]]; then
    XC_OS=$(go env GOOS)
    XC_ARCH=$(go env GOARCH)
fi

if ! which gox > /dev/null; then
    echo "==> Installing gox..."
    go install github.com/mitchellh/gox
fi

# Instruct gox to build statically linked binaries
export CGO_ENABLED=0

# Set module download mode to readonly to not implicitly update go.mod
export GOFLAGS="-mod=readonly"

# In release mode we don't want debug information in the binary
if [[ -n "${TF_RELEASE}" ]]; then
    LD_FLAGS="-s -w"
fi

# Ensure all remote modules are downloaded and cached before build so that
# the concurrent builds launched by gox won't race to redundantly download them.
go mod download

# Build!
echo "==> Building..."
gox \
    -os="${XC_OS}" \
    -arch="${XC_ARCH}" \
    -osarch="${XC_EXCLUDE_OSARCH}" \
    -ldflags "${LD_FLAGS}" \
    -output "pkg/{{.OS}}_{{.Arch}}/terraform" \
    .

# Move all the compiled things to the $GOPATH/bin
GOPATH=${GOPATH:-$(go env GOPATH)}
case $(uname) in
    CYGWIN*)
        GOPATH="$(cygpath $GOPATH)"
        ;;
esac
OLDIFS=$IFS
IFS=: MAIN_GOPATH=($GOPATH)
IFS=$OLDIFS

# Create GOPATH/bin if it's doesn't exists
if [ ! -d $MAIN_GOPATH/bin ]; then
    echo "==> Creating GOPATH/bin directory..."
    mkdir -p $MAIN_GOPATH/bin
fi

# Copy our OS/Arch to the bin/ directory
DEV_PLATFORM="./pkg/$(go env GOOS)_$(go env GOARCH)"
if [[ -d "${DEV_PLATFORM}" ]]; then
    for F in $(find ${DEV_PLATFORM} -mindepth 1 -maxdepth 1 -type f); do
        cp ${F} bin/
        cp ${F} ${MAIN_GOPATH}/bin/
    done
fi

if [ "${TF_DEV}x" = "x" ]; then
    # Zip and copy to the dist dir
    echo "==> Packaging..."
    for PLATFORM in $(find ./pkg -mindepth 1 -maxdepth 1 -type d); do
        OSARCH=$(basename ${PLATFORM})
        echo "--> ${OSARCH}"

        pushd $PLATFORM >/dev/null 2>&1
        zip ../${OSARCH}.zip ./*
        popd >/dev/null 2>&1
    done
fi

# Done!
echo
echo "==> Results:"
ls -hl bin/
