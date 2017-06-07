---
layout: "github"
page_title: "GitHub: github_branch_protection"
sidebar_current: "docs-github-resource-branch-protection"
description: |-
  Protects a GitHub branch.
---

# github\_branch\_protection

Protects a GitHub branch.

This resource allows you to configure branch protection for repositories in your organization. When applied, the branch will be protected from forced pushes and deletion. Additional constraints, such as required status checks or restrictions on users and teams, can also be configured.

## Example Usage

```
# Protect the master branch of the foo repository. Additionally, require that
# the "ci/travis" context to be passing and only allow the engineers team merge
# to the branch.
resource "github_branch_protection" "foo_master" {
  repository = "foo"
  branch = "master"

  required_status_checks {
    include_admins = true
    strict = false
    contexts = ["ci/travis"]
  }

  required_pull_request_reviews {
    include_admins = true
  }

  restrictions {
    teams = ["engineers"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `repository` - (Required) The GitHub repository name.
* `branch` - (Required) The Git branch to protect.
* `required_status_checks` - (Optional) Enforce restrictions for required status checks. See [Required Status Checks](#required-status-checks) below for details.
* `required_pull_request_reviews` - (Optional) Enforce restrictions for pull request reviews. See [Required Pull Request Reviews](#required-pull-request-reviews) below for details.
* `restrictions` - (Optional) Enforce restrictions for the users and teams that may push to the branch. See [Restrictions](#restrictions) below for details.

### Required Status Checks

`required_status_checks` supports the following arguments:

* `include_admins`: (Optional) Enforce required status checks for repository administrators. Defaults to `false`.
* `strict`: (Optional) Require branches to be up to date before merging. Defaults to `false`.
* `contexts`: (Optional) The list of status checks to require in order to merge into this branch. No status checks are required by default.

### Required Pull Request Reviews

`required_pull_request_reviews` supports the following arguments:

* `include_admins`: (Optional) Enforce required status checks for repository administrators. Defaults to `false`.

### Restrictions

`restrictions` supports the following arguments:

* `users`: (Optional) The list of user logins with push access.
* `teams`: (Optional) The list of team slugs with push access.

`restrictions` is only available for organization-owned repositories.

## Import

Github Branch Protection can be imported using an id made up of `repository:branch`, e.g.

```
$ terraform import github_branch_protection.terraform terraform:master
```