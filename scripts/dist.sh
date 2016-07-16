#!/usr/bin/env bash
set -e

# Get the version from the command line
VERSION=$1
if [ -z $VERSION ]; then
    echo "Please specify a version."
    exit 1
fi

# Make sure we have a bintray API key
if [[ -z $AWS_ACCESS_KEY_ID  || -z $AWS_SECRET_ACCESS_KEY ]]; then
    echo "Please set AWS access keys as env vars before running this script."
    exit 1
fi

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that
cd $DIR

# Zip all the files
rm -rf ./pkg/dist
mkdir -p ./pkg/dist
for FILENAME in $(find ./pkg -mindepth 1 -maxdepth 1 -type f); do
    FILENAME=$(basename $FILENAME)
    cp ./pkg/${FILENAME} ./pkg/dist/terraform_${VERSION}_${FILENAME}
done

# Make the checksums
echo "==> Signing..."
pushd ./pkg/dist
rm -f ./terraform_${VERSION}_SHA256SUMS*
shasum -a256 * > ./terraform_${VERSION}_SHA256SUMS
gpg --default-key 348FFC4C --detach-sig ./terraform_${VERSION}_SHA256SUMS
popd

# Upload
hc-releases upload ./pkg/dist

exit 0
