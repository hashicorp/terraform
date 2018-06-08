#!/bin/bash

# For normal use this package can just be tested with "go test" as standard,
# but this script is an alternative to allow the tests to be run somewhere
# other than where they are built.

# The primary use for this is cross-compilation, where e.g. we can produce an
# archive that can be extracted on a Windows system to run the e2e tests there:
#    $ GOOS=windows GOARCH=amd64 ./make-archive.sh
#
# This will produce a zip file build/terraform-s2stest_windows_amd64.zip which
# can be shipped off to a Windows amd64 system, extracted to some directory,
# and then executed as follows:
#    set TF_ACC=1
#    ./e2etest.exe
# Since the test archive includes both the test fixtures and the compiled
# terraform executable along with this test program, the result is
# self-contained and does not require a local Go compiler on the target system.

set +euo pipefail

# Always run from the directory where this script lives
cd "$( dirname "${BASH_SOURCE[0]}" )"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"
GOEXE="$(go env GOEXE)"
OUTDIR="build/${GOOS}_${GOARCH}"
OUTFILE="terraform-e2etest_${GOOS}_${GOARCH}.zip"

mkdir -p "$OUTDIR"

# We need the test fixtures available when we run the tests.
cp -r test-fixtures "$OUTDIR/test-fixtures"

# Bundle a copy of our binary so the target system doesn't need the go
# compiler installed.
go build -o "$OUTDIR/terraform$GOEXE" github.com/hashicorp/terraform

# Build the test program
go test -o "$OUTDIR/e2etest$GOEXE" -c -ldflags "-X github.com/hashicorp/terraform/command/e2etest.terraformBin=./terraform$GOEXE" github.com/hashicorp/terraform/command/e2etest

# Now bundle it all together for easy shipping!
cd "$OUTDIR"
zip -r "../$OUTFILE" *

echo "e2etest archive created at build/$OUTFILE"
