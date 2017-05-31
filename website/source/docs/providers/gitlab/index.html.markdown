---
layout: "gitlab"
page_title: "Provider: GitLab"
sidebar_current: "docs-gitlab-index"
description: |-
  The GitLab provider is used to interact with GitLab group or user resources.
---

# GitLab Provider

The GitLab provider is used to interact with GitLab group or user resources.

It needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the GitLab Provider
provider "gitlab" {
    token = "${var.gitlab_token}"
}

# Add a project owned by the user
resource "gitlab_project" "sample_project" {
    name = "example"
}

# Add a hook to the project
resource "gitlab_project_hook" "sample_project_hook" {
    project = "${gitlab_project.sample_project.id}"
    url = "https://example.com/project_hook"
}

# Add a deploy key to the project
resource "gitlab_deploy_key" "sample_deploy_key" {
    project = "${gitlab_project.sample_project.id}"
    title = "terraform example"
    key = "ssh-rsa AAAA..."
}

# Add a group
resource "gitlab_group" "sample_group" {
    name = "example"
    path = "example"
    description = "An example group"
}

# Add a project to the group - example/example
resource "gitlab_project" "sample_group_project" {
    name = "example"
    namespace_id = "${gitlab_group.sample_group.id}"
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `token` - (Optional) This is the GitLab personal access token. It must be provided, but
  it can also be sourced from the `GITLAB_TOKEN` environment variable.

* `base_url` - (Optional) This is the target GitLab base API endpoint. Providing a value is a
  requirement when working with GitLab CE or GitLab Enterprise e.g. https://my.gitlab.server/api/v3/.
  It is optional to provide this value and it can also be sourced from the `GITLAB_BASE_URL` environment variable.
  The value must end with a slash.
