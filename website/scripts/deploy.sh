#!/bin/bash
set -e

PROJECT="terraform"
PROJECT_URL="www.terraform.io"
FASTLY_SERVICE_ID="7GrxRJP3PVBuqQbyxYQ0MV"

# Ensure the proper AWS environment variables are set
if [ -z "$AWS_ACCESS_KEY_ID" ]; then
  echo "Missing AWS_ACCESS_KEY_ID!"
  exit 1
fi

if [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
  echo "Missing AWS_SECRET_ACCESS_KEY!"
  exit 1
fi

# Ensure the proper Fastly keys are set
if [ -z "$FASTLY_API_KEY" ]; then
  echo "Missing FASTLY_API_KEY!"
  exit 1
fi

# Ensure we have s3cmd installed
if ! command -v "s3cmd" >/dev/null 2>&1; then
  echo "Missing s3cmd!"
  exit 1
fi

# Get the parent directory of where this script is and change into our website
# directory
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$(cd -P "$( dirname "$SOURCE" )/.." && pwd)"

# Delete any .DS_Store files for our OS X friends.
find "$DIR" -type f -name '.DS_Store' -delete

# Upload the files to S3 - we disable mime-type detection by the python library
# and just guess from the file extension because it's surprisingly more
# accurate, especially for CSS and javascript. We also tag the uploaded files
# with the proper Surrogate-Key, which we will later purge in our API call to
# Fastly.
if [ -z "$NO_UPLOAD" ]; then
  echo "Uploading to S3..."

  # Check that the site has been built
  if [ ! -d "$DIR/build" ]; then
    echo "Missing compiled website! Run 'make build' to compile!"
    exit 1
  fi

  s3cmd \
    --quiet \
    --delete-removed \
    --guess-mime-type \
    --no-mime-magic \
    --acl-public \
    --recursive \
    --add-header="Cache-Control: max-age=31536000" \
    --add-header="x-amz-meta-surrogate-key: site-$PROJECT" \
    sync "$DIR/build/" "s3://hc-sites/$PROJECT/latest/"

  # The s3cmd guessed mime type for text files is often wrong. This is
  # problematic for some assets, so force their mime types to be correct.
  echo "Overriding javascript mime-types..."
  s3cmd \
    --mime-type="application/javascript" \
    --exclude "*" \
    --include "*.js" \
    --recursive \
    modify "s3://hc-sites/$PROJECT/latest/"

  echo "Overriding css mime-types..."
  s3cmd \
    --mime-type="text/css" \
    --exclude "*" \
    --include "*.css" \
    --recursive \
    modify "s3://hc-sites/$PROJECT/latest/"

  echo "Overriding svg mime-types..."
  s3cmd \
    --mime-type="image/svg+xml" \
    --exclude "*" \
    --include "*.svg" \
    --recursive \
    modify "s3://hc-sites/$PROJECT/latest/"
fi

# Perform a soft-purge of the surrogate key.
if [ -z "$NO_PURGE" ]; then
  echo "Purging Fastly cache..."
  curl \
    --fail \
    --silent \
    --output /dev/null \
    --request "POST" \
    --header "Accept: application/json" \
    --header "Fastly-Key: $FASTLY_API_KEY" \
    --header "Fastly-Soft-Purge: 1" \
    "https://api.fastly.com/service/$FASTLY_SERVICE_ID/purge/site-$PROJECT"
fi

# Warm the cache with recursive wget.
if [ -z "$NO_WARM" ]; then
  echo "Warming Fastly cache..."
  wget \
    --recursive \
    --delete-after \
    --quiet \
    "https://$PROJECT_URL/"
fi
