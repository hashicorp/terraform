---
layout: "github"
page_title: "GitHub: github_repository"
sidebar_current: "docs-github-resource-repository"
description: |-
  Creates and manages repositories within Github organizations
---

# github_repository

This resource allows you to create and manage repositories within your
Github organization.

This resource cannot currently be used to manage *personal* repositories,
outside of organizations.

## Example Usage

```hcl
resource "github_repository" "example" {
  name        = "example"
  description = "My awesome codebase"

  private = true
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the repository.

* `description` - (Optional) A description of the repository.

* `homepage_url` - (Optional) URL of a page describing the project.

* `private` - (Optional) Set to `true` to create a private repository.
  Repositories are created as public (e.g. open source) by default.

* `has_issues` - (Optional) Set to `true` to enable the Github Issues features
  on the repository.

* `has_wiki` - (Optional) Set to `true` to enable the Github Wiki features on
  the repository.

* `has_downloads` - (Optional) Set to `true` to enable the (deprecated)
  downloads features on the repository.

* `auto_init` - (Optional) Meaningful only during create; set to `true` to
  produce an initial commit in the repository.

## Attributes Reference

The following additional attributes are exported:

* `full_name` - A string of the form "orgname/reponame".

* `default_branch` - The name of the repository's default branch.

* `ssh_clone_url` - URL that can be provided to `git clone` to clone the
  repository via SSH.

* `http_clone_url` - URL that can be provided to `git clone` to clone the
  repository via HTTPS.

* `git_clone_url` - URL that can be provided to `git clone` to clone the
  repository anonymously via the git protocol.

* `svn_url` - URL that can be provided to `svn checkout` to check out
  the repository via Github's Subversion protocol emulation.
  

## Import

Repositories can be imported using the `name`, e.g.

```
$ terraform import github_repository.terraform terraform
```
