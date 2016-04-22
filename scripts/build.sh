#!/usr/bin/env bash
#
# This script builds the application from source for multiple platforms.

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that directory
cd "$DIR"

# Get the git commit
GIT_COMMIT=$(git rev-parse HEAD)
GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)

# Determine the arch/os combos we're building for
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:-linux darwin windows freebsd openbsd solaris}

# Delete the old dir
echo "==> Removing old directory..."
rm -f bin/*
rm -rf pkg/*
mkdir -p bin/

# If its dev mode, only build for ourself
if [ "${TF_DEV}x" != "x" ]; then
    XC_OS=$(go env GOOS)
    XC_ARCH=$(go env GOARCH)
fi

if ! which gox > /dev/null; then
    echo "==> Installing gox..."
    go get -u github.com/mitchellh/gox
fi

LD_FLAGS="-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY}"
# In relase mode we don't want debug information in the binary
if [[ -n "${TF_RELEASE}" ]]; then
    LD_FLAGS="-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -s -w"
fi

# Build!
echo "==> Building..."
gox \
    -os="${XC_OS}" \
    -arch="${XC_ARCH}" \
    -ldflags "${LD_FLAGS}" \
    -output "pkg/{{.OS}}_{{.Arch}}/terraform-{{.Dir}}" \
    $(go list ./... | grep -v /vendor/)

# Make sure "terraform-terraform" is renamed properly
for PLATFORM in $(find ./pkg -mindepth 1 -maxdepth 1 -type d); do
    set +e
    mv ${PLATFORM}/terraform-terraform.exe ${PLATFORM}/terraform.exe 2>/dev/null
    mv ${PLATFORM}/terraform-terraform ${PLATFORM}/terraform 2>/dev/null
    set -e
done

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
for F in $(find ${DEV_PLATFORM} -mindepth 1 -maxdepth 1 -type f); do
    cp ${F} bin/
    cp ${F} ${MAIN_GOPATH}/bin/
done

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
