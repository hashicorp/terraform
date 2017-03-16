---
title: "Git Integration"
---

# Git Integation

Git repositories can be integrated with Atlas by using
[`terraform push`](https://www.terraform.io/docs/commands/push.html) to import
Terraform configuration when changes are committed. When Terraform
configuration is imported using `terraform push` a plan is automatically queued
in Atlas.

_**Note:** This integration is for Git repositories **not** hosted on GitHub. 
For repositories on GitHub, there is native [GitHub Integration](/help/terraform/vcs/github).

## Setup

Terraform configuration can be manually imported by running `terraform push`
like below:

```
$ terraform push -name=$ATLAS_USERNAME/ENV_NAME
```

A better option than having to manually run `terraform push` is to run it
using a git commit hook. A client-side `pre-push` hook is suitable and will
push your Terraform configuration when you push local changes to your Git
server.

### Client-side Commit Hook

The script below will execute `terraform push` when you push local changes to
your Git server. Place the script at `.git/pre-push` in your local Git
repository, set the necessary variables, and ensure the script is executable.

```
#!/bin/bash
#
# An example hook script to push Terraform configuration to Atlas.
#
# Set the following variables for your project:
# - ENV_NAME - your Atlas environment name (e.g. org/env)
# - TERRAFORM_DIR - the local directory to push
# - DEFAULT_BRANCH - the branch to push. Other branches will be ignored.

ENV_NAME="YOUR_ORG/YOUR_ENV"
TERRAFORM_DIR="terraform"
DEFAULT_BRANCH=""

if [[ -z "$ENV_NAME" || -z "$TERRAFORM_DIR" || -z "$DEFAULT_BRANCH" ]]; then
  echo 'pre-push hook: One or more variables are undefined. Canceling push.'
  exit 1
fi

current_branch=$(git symbolic-ref HEAD | sed -e 's,.*/\(.*\),\1,')

if [ "$current_branch" == "$DEFAULT_BRANCH" ]; then
  echo "pre-push hook: Pushing branch [$current_branch] to Atlas environment [$ENV_NAME]."
  terraform push -name="$ENV_NAME" $TERRAFORM_DIR
else
  echo "pre-push hook: NOT pushing branch [$current_branch] to Atlas environment [$ENV_NAME]."
fi

```
