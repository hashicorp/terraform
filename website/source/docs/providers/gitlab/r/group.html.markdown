---
layout: "gitlab"
page_title: "GitLab: gitlab_group"
sidebar_current: "docs-gitlab-resource-group"
description: |-
  Creates and manages GitLab groups
---

# gitlab\_group

This resource allows you to create and manage GitLab groups.
Note your provider will need to be configured with admin-level access for this resource to work.

## Example Usage

```hcl
resource "gitlab_group" "example" {
  name        = "example"
  path        = "example"
  description = "An example group"
}

// Create a project in the example group
resource "gitlab_project" "example" {
  name         = "example"
  description  = "An example project"
  namespace_id = "${gitlab_group.example.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of this group.

* `path` - (Required) The url of the hook to invoke.

* `description` - (Optional) The description of the group.

* `lfs_enabled` - (Optional) Boolean, defaults to true.  Whether to enable LFS
support for projects in this group.

* `request_access_enabled` - (Optional) Boolean, defaults to false.  Whether to
enable users to request access to the group.

* `visibility_level` - (Optional) Set to `public` to create a public group.
  Valid values are `private`, `internal`, `public`.
  Groups are created as private by default.

## Attributes Reference

The resource exports the following attributes:

* `id` - The unique id assigned to the group by the GitLab server.  Serves as a
  namespace id where one is needed.
