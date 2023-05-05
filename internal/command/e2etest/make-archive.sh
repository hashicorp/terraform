#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


# For normal use this package can just be tested with "go test" as standard,
# but this script is an alternative to allow the tests to be run somewhere
# other than where they are built.

# The primary use for this is cross-compilation, where e.g. we can produce an
# archive that can be extracted on a Windows system to run the e2e tests there:
#    $ GOOS=windows GOARCH=amd64 ./make-archive.sh
#
# This will produce a zip file build/terraform-e2etest_windows_amd64.zip which
# can be shipped off to a Windows amd64 system, extracted to some directory,
# and then executed as follows:
#    set TF_ACC=1
#    ./e2etest.exe
#
# Because separated e2etest harnesses are intended for testing against "real"
# release executables, the generated archives don't include a copy of
# the Terraform executable. Instead, the caller of the tests must retrieve
# and extract a release package into the working directory before running
# the e2etest executable, so that "e2etest" can find and execute it.

set +euo pipefail

# Always run from the directory where this script lives
cd "$( dirname "${BASH_SOURCE[0]}" )"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"
GOEXE="$(go env GOEXE)"
OUTDIR="build/${GOOS}_${GOARCH}"
OUTFILE="terraform-e2etest_${GOOS}_${GOARCH}.zip"

LDFLAGS="-X github.com/hashicorp/terraform/internal/command/e2etest.terraformBin=./terraform$GOEXE"
# Caller may pass in the environment variable GO_LDFLAGS with additional
# flags we'll use when building.
if [ -n "${GO_LDFLAGS+set}" ]; then
    LDFLAGS="${GO_LDFLAGS} ${LDFLAGS}"
fi

mkdir -p "$OUTDIR"

# We need the test fixtures available when we run the tests.
cp -r testdata "$OUTDIR/testdata"

# Build the test program
go test -o "$OUTDIR/e2etest$GOEXE" -c -ldflags "$LDFLAGS" github.com/hashicorp/terraform/internal/command/e2etest

# Now bundle it all together for easy shipping!
cd "$OUTDIR"
zip -r "../$OUTFILE" *

echo "e2etest archive created at build/$OUTFILE"
