---
layout: "github"
page_title: "Provider: Github"
sidebar_current: "docs-github-index"
description: |-
  The Github provider is used to interact with Github organization resources.
---

# Github Provider

The Github provider is used to interact with Github organization resources. 

The provider allows you to manage your Github organization's members and teams easily. 
It needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Github Provider
provider "github" {
    token = "${var.github_token}"
    organization = "${var.github_organization}"
}

# Add a user to the organization
resource "github_membership" "membership_for_user_x" {
    ...
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `token` - (Optional) This is the Github personal access token. It must be provided, but
  it can also be sourced from the `GITHUB_TOKEN` environment variable.

* `organization` - (Optional) This is the target Github organization to manage. The account
  corresponding to the token will need "owner" privileges for this organization. It must be provided, but
  it can also be sourced from the `GITHUB_ORGANIZATION` environment variable.
