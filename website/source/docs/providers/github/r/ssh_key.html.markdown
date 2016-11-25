---
layout: "github"
page_title: "GitHub: github_ssh_key"
sidebar_current: "docs-github-resource-ssh-key"
description: |-
  Provides a GitHub ssh key resource.
---

# github\_ssh_key

Provides a GitHub ssh key resource.

This resource allows you to add/remove ssh key to the user account. When applied,
sshkey will be added into user's account with a given title

## Example Usage

```
# Add a user to the organization
resource "github_repository_sshkey" "some_key" {
    title = "ssh_key title"
    sshkey = "ssh public key"
}
```

## Argument Reference

The following arguments are supported:

* `title` - (Required) The title to add to the user's account.
* `sshkey` - (Required) Public ssh key that will be added to the user's account.
