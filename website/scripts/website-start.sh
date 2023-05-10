# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

######################################################
# NOTE: This file is managed by the Digital Team's   #
# Terraform configuration @ hashicorp/mktg-terraform #
######################################################

# Repo which we are cloning and executing npm run build:deploy-preview within
REPO_TO_CLONE=dev-portal
# Set the subdirectory name for the dev-portal app
PREVIEW_DIR=website-preview
# The product for which we are building the deploy preview
PRODUCT=terraform
# Preview mode, controls the UI rendered (either the product site or developer). Can be `io` or `developer`
PREVIEW_MODE=developer

# Get the git branch of the commit that triggered the deploy preview
# This will power remote image assets in local and deploy previews
CURRENT_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# This is where content files live, relative to the website-preview dir. If omitted, "../content" will be used
LOCAL_CONTENT_DIR=../docs

should_pull=true

# Clone the dev-portal project, if needed
if [ ! -d "$PREVIEW_DIR" ]; then
    echo "⏳ Cloning the $REPO_TO_CLONE repo, this might take a while..."
    git clone --depth=1 https://github.com/hashicorp/$REPO_TO_CLONE.git "$PREVIEW_DIR"
    should_pull=false
fi

cd "$PREVIEW_DIR"

# If the directory already existed, pull to ensure the clone is fresh
if [ "$should_pull" = true ]; then
    git pull origin main
fi

# Run the dev-portal content-repo start script
REPO=$PRODUCT \
PREVIEW_FROM_REPO=$PRODUCT \
LOCAL_CONTENT_DIR=$LOCAL_CONTENT_DIR \
CURRENT_GIT_BRANCH=$CURRENT_GIT_BRANCH \
PREVIEW_MODE=$PREVIEW_MODE \
npm run start:local-preview