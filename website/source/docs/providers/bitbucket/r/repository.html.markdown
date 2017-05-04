---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_repository"
sidebar_current: "docs-bitbucket-resource-repository"
description: |-
  Provides a Bitbucket Repository
---

# bitbucket\_repository

Provides a Bitbucket repository resource.

This resource allows you manage your repositories such as scm type, if it is
private, how to fork the repository and other options.

## Example Usage

```hcl
# Manage your repository
resource "bitbucket_repository" "infrastructure" {
  owner = "myteam"
  name  = "terraform-code"
}
```

## Argument Reference

The following arguments are supported:

* `owner` - (Required) The owner of this repository. Can be you or any team you
  have write access to.
* `name` - (Optional) The name of the repository.
* `scm` - (Optional) What SCM you want to use. Valid options are hg or git.
  Defaults to git.
* `is_private` - (Optional) If this should be private or not. Defaults to `true`.
* `website` - (Optional) URL of website associated with this repository.
* `language` - (Optional) What the language of this repository should be.
* `has_issues` - (Optional) If this should have issues turned on or not.
* `has_wiki` - (Optional) If this should have wiki turned on or not.
* `project_key` - (Optional) If you want to have this repo associated with a
  project.
* `fork_policy` - (Optional) What the fork policy should be. Defaults to
  allow_forks.
* `description` - (Optional) What the description of the repo is.

## Computed Arguments

The following arguments are computed. You can access both `clone_ssh` and
`clone_https` for getting a clone URL.

## Import

Repositories can be imported using the `name`, e.g.

```
$ terraform import bitbucket_repository.my-repo my-repo
```
