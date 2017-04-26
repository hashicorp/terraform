---
layout: "gitlab"
page_title: "GitLab: gitlab_project"
sidebar_current: "docs-gitlab-resource-project"
description: |-
  Creates and manages projects within Github organizations
---

# gitlab\_project

This resource allows you to create and manage projects within your
GitLab organization.


## Example Usage

```hcl
resource "gitlab_repository" "example" {
  name        = "example"
  description = "My awesome codebase"

  visbility_level = "public"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the project.

* `description` - (Optional) A description of the project.

* `default_branch` - (Optional) The default branch for the project.

* `issues_enabled` - (Optional) Enable issue tracking for the project.

* `merge_requests_enabled` - (Optional) Enable merge requests for the project.

* `wiki_enabled` - (Optional) Enable wiki for the project.

* `snippets_enabled` - (Optional) Enable snippets for the project.

* `visbility_level` - (Optional) Set to `public` to create a public project.
  Valid values are `private`, `internal`, `public`.
  Repositories are created as private by default.

## Attributes Reference

The following additional attributes are exported:

* `ssh_url_to_repo` - URL that can be provided to `git clone` to clone the
  repository via SSH.

* `http_url_to_repo` - URL that can be provided to `git clone` to clone the
  repository via HTTP.

* `web_url` - URL that can be used to find the project in a browser.
