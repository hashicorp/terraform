---
layout: "gitlab"
page_title: "GitLab: gitlab_deploy_key"
sidebar_current: "docs-gitlab-resource-deploy_key"
description: |-
  Creates and manages deploy keys for GitLab projects
---

# gitlab\_deploy\_key

This resource allows you to create and manage deploy keys for your GitLab projects.


## Example Usage

```hcl
resource "gitlab_deploy_key" "example" {
  project = "example/deploying"
  title   = "Example deploy key"
  key     = "ssh-rsa AAAA..."
}
```

## Argument Reference

The following arguments are supported:

* `project` - (Required, string) The name or id of the project to add the deploy key to.

* `title` - (Required, string) A title to describe the deploy key with.

* `key` - (Required, string) The public ssh key body.

* `can_push` - (Optional, boolean) Allow this deploy key to be used to push changes to the project.  Defaults to `false`. **NOTE::** this cannot currently be managed.
