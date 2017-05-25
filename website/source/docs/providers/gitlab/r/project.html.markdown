---
layout: "gitlab"
page_title: "GitLab: gitlab_project"
sidebar_current: "docs-gitlab-resource-project-x"
description: |-
  Creates and manages projects within GitLab groups or within your user
---

# gitlab\_project

This resource allows you to create and manage projects within your
GitLab group or within your user.


## Example Usage

```hcl
resource "gitlab_project" "example" {
  name        = "example"
  description = "My awesome codebase"

  visibility_level = "public"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the project.

* `namespace_id` - (Optional) The namespace (group or user) of the project. Defaults to your user.
  See [`gitlab_group`](group.html) for an example.

* `description` - (Optional) A description of the project.

* `default_branch` - (Optional) The default branch for the project.

* `issues_enabled` - (Optional) Enable issue tracking for the project.

* `merge_requests_enabled` - (Optional) Enable merge requests for the project.

* `wiki_enabled` - (Optional) Enable wiki for the project.

* `snippets_enabled` - (Optional) Enable snippets for the project.

* `visibility_level` - (Optional) Set to `public` to create a public project.
  Valid values are `private`, `internal`, `public`.
  Repositories are created as private by default.

## Attributes Reference

The following additional attributes are exported:

* `id` - Integer that uniquely identifies the project within the gitlab install.

* `ssh_url_to_repo` - URL that can be provided to `git clone` to clone the
  repository via SSH.

* `http_url_to_repo` - URL that can be provided to `git clone` to clone the
  repository via HTTP.

* `web_url` - URL that can be used to find the project in a browser.
