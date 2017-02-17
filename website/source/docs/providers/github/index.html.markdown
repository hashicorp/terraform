---
layout: "github"
page_title: "Provider: GitHub"
sidebar_current: "docs-github-index"
description: |-
  The GitHub provider is used to interact with GitHub organization resources.
---

# GitHub Provider

The GitHub provider is used to interact with GitHub organization resources.

The provider allows you to manage your GitHub organization's members and teams easily.
It needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the GitHub Provider
provider "github" {
    token = "${var.github_token}" # will be deprecated use organization_key instead of token
    organization = "${var.github_organization}"
    user_key = "${var.github_user_key}"
    organization_key = "${var.github_organization_key}"
}

# Add a user to the organization
resource "github_membership" "membership_for_user_x" {
    ...
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `token` - (Optional) This is the GitHub personal access token. It must be provided, but
  it can also be sourced from the `GITHUB_TOKEN` environment variable.(This token will be deprecated, use organization_key instead of token)

* `organization` - (Optional) This is the target GitHub organization to manage. The account
  corresponding to the token will need "owner" privileges for this organization. It must be provided, but
  it can also be sourced from the `GITHUB_ORGANIZATION` environment variable.

* `base_url` - (Optional) This is the target GitHub base API endpoint. Providing a value is a
  requirement when working with GitHub Enterprise.  It is optional to provide this value and
  it can also be sourced from the `GITHUB_BASE_URL` environment variable.  The value must end with a slash.

* `user_key` - (Optional) This is the GitHub personal access token. It can also be sourced
  from the `GITHUB_USER_KEY` environment variable. It allows you to accept memberships without invitations & add sshkeys into user account & fork and repo into user's account

* `organization_key` - (Optional) This is the GitHub personal access token for the
  owner/admin  organization. It must be provided, but
  it can also be sourced from the `GITHUB_ORGANIZATION_KEY` environment variable.
