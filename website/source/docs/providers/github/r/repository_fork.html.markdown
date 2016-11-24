---
layout: "github"
page_title: "GitHub: github_repository_fork"
sidebar_current: "docs-github-resource-repository_fork"
description: |-
  Provides a GitHub repository fork resource.
---

# github\_repository_fork

Provides a GitHub repository fork resource.

This resource allows you to create/remove fork any repository of organization. When applied,
repository will be forked into the user's account

## Example Usage

```
# Add a user to the organization
resource "github_membership" "membership_for_some_user" {
    owner = "owner of the repository"
    repository = "repository to fork"
    organization = "optional parameter to fork into organization"
}
```

## Argument Reference

The following arguments are supported:

* `owner` - (Required) Owner of the repository that will be forked .
* `repository` - (Required) Repository that will be forked.
* `organization` - (Optional) organization specifies the optional parameter to fork the repository into the organization.
