---
layout: "gitlab"
page_title: "Provider: GitLab"
sidebar_current: "docs-gitlab-index"
description: |-
  The GitLab provider is used to interact with GitLab organization resources.
---

# GitLab Provider

The GitLab provider is used to interact with GitLab organization resources.

The provider allows you to manage your GitLab organization's members and teams easily.
It needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the GitLab Provider
provider "gitlab" {
    token = "${var.github_token}"
}

# Add a project to the organization
resource "gitlab_project" "sample_project" {
    ...
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `token` - (Optional) This is the GitLab personal access token. It must be provided, but
  it can also be sourced from the `GITLAB_TOKEN` environment variable.

* `base_url` - (Optional) This is the target GitLab base API endpoint. Providing a value is a
  requirement when working with GitLab CE or GitLab Enterprise.  It is optional to provide this value and
  it can also be sourced from the `GITLAB_BASE_URL` environment variable.  The value must end with a slash.
