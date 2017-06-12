---
layout: "github"
page_title: "GitHub: github_team_repository"
sidebar_current: "docs-github-resource-team-repository"
description: |-
  Manages the associations between teams and repositories.
---

# github_team_repository

This resource manages relationships between teams and repositories
in your Github organization.

Creating this resource grants a particular team permissions on a
particular repository.

The repository and the team must both belong to the same organization
on Github. This resource does not actually *create* any repositories;
to do that, see [`github_repository`](repository.html).

## Example Usage

```hcl
# Add a repository to the team
resource "github_team" "some_team" {
  name        = "SomeTeam"
  description = "Some cool team"
}

resource "github_repository" "some_repo" {
  name = "some-repo"
}

resource "github_team_repository" "some_team_repo" {
  team_id    = "${github_team.some_team.id}"
  repository = "${github_repository.some_repo.name}"
  permission = "pull"
}
```

## Argument Reference

The following arguments are supported:

* `team_id` - (Required) The GitHub team id
* `repository` - (Required) The repository to add to the team.
* `permission` - (Optional) The permissions of team members regarding the repository.
  Must be one of `pull`, `push`, or `admin`. Defaults to `pull`.


## Import

Github Team Membership can be imported using an id made up of `teamid:repository`, e.g.

```
$ terraform import github_team_repository.terraform_repo 1234567:terraform
```