#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

# This script builds the application from source for multiple platforms.

# Resolve the script's directory
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do
    SOURCE="$(readlink "$SOURCE")"
done
DIR="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"

# Navigate to the script's directory
cd "$DIR" || exit 1

# Determine the architectures and operating systems to build for
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:-"linux darwin windows freebsd openbsd solaris"}
XC_EXCLUDE_OSARCH="!darwin/arm !darwin/386"

# Clean up old directories
echo "==> Removing old directories..."
rm -rf bin/* pkg/*
mkdir -p bin/

# Use dev mode settings if TF_DEV is set
if [[ -n "$TF_DEV" ]]; then
    XC_OS=$(go env GOOS)
    XC_ARCH=$(go env GOARCH)
fi

# Install gox if not available
if ! command -v gox &>/dev/null; then
    echo "==> Installing gox..."
    go install github.com/mitchellh/gox
fi

# Configure Go environment
export CGO_ENABLED=0
export GOFLAGS="-mod=readonly"

# Configure release mode settings
LD_FLAGS=""
if [[ -n "$TF_RELEASE" ]]; then
    LD_FLAGS="-s -w -X 'github.com/hashicorp/terraform/version.dev=no'"
fi

# Pre-download all modules to prevent concurrency issues
go mod download

# Build the binaries
echo "==> Building..."
gox \
    -os="$XC_OS" \
    -arch="$XC_ARCH" \
    -osarch="$XC_EXCLUDE_OSARCH" \
    -ldflags "$LD_FLAGS" \
    -output "pkg/{{.OS}}_{{.Arch}}/terraform" \
    .

# Determine GOPATH
GOPATH=${GOPATH:-$(go env GOPATH)}
case $(uname) in
    CYGWIN*) GOPATH="$(cygpath "$GOPATH")" ;;
esac

# Get the main GOPATH directory
IFS=: read -r MAIN_GOPATH _ <<< "$GOPATH"

# Ensure GOPATH/bin exists
if [[ ! -d "$MAIN_GOPATH/bin" ]]; then
    echo "==> Creating GOPATH/bin directory..."
    mkdir -p "$MAIN_GOPATH/bin"
fi

# Copy built binaries to bin/ and GOPATH/bin
DEV_PLATFORM="./pkg/$(go env GOOS)_$(go env GOARCH)"
if [[ -d "$DEV_PLATFORM" ]]; then
    for FILE in "$DEV_PLATFORM"/*; do
        cp "$FILE" bin/
        cp "$FILE" "$MAIN_GOPATH/bin/"
    done
fi

# Package binaries for non-dev builds
if [[ -z "$TF_DEV" ]]; then
    echo "==> Packaging..."
    for PLATFORM in ./pkg/*; do
        if [[ -d "$PLATFORM" ]]; then
            OSARCH=$(basename "$PLATFORM")
            echo "--> $OSARCH"
            pushd "$PLATFORM" >/dev/null
            zip -r "../${OSARCH}.zip" ./*
            popd >/dev/null
        fi
    done
fi

# Display results
echo
echo "==> Results:"
ls -hl bin/
