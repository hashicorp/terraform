---
layout: "github"
page_title: "GitHub: github_membership"
sidebar_current: "docs-github-resource-membership"
description: |-
  Provides a GitHub membership resource.
---

# github\_membership

Provides a GitHub membership resource.

This resource allows you to add/remove users from your organization. When applied,
an invitation will be sent to the user to become part of the organization. When
destroyed, either the invitation will be cancelled or the user will be removed.

## Example Usage

```
# Add a user to the organization
resource "github_membership" "membership_for_some_user" {
    username = "SomeUser"
    role = "member"
}
```

## Argument Reference

The following arguments are supported:

* `username` - (Required) The user to add to the organization.
* `role` - (Optional) The role of the user within the organization. 
            Must be one of `member` or `admin`. Defaults to `member`.
