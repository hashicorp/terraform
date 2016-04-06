---
layout: "github"
page_title: "Github: github_team_repository"
sidebar_current: "docs-github-resource-team-repository"
description: |-
  Provides a Github team repository resource.
---

# github\_team_repository

Provides a Github team repository resource.

This resource allows you to add/remove repositories from teams in your organization. When applied,
the repository will be added to the team. When destroyed, the repository will be removed from the team.

## Example Usage

```
# Add a repository to the team
resource "github_team" "some_team" {
    name = "SomeTeam"
    description = "Some cool team"
}

resource "github_team_repository" "some_team_repo" {
	team_id = "${github_team.some_team.id}"
	repository = "our-repo"
	permission = "pull"
}
```

## Argument Reference

The following arguments are supported:

* `team_id` - (Required) The Github team id
* `repository` - (Required) The repository to add to the team.
* `permission` - (Optional) The permissions of team members regarding the repository. 
                  Must be one of `pull`, `push`, or `admin`. Defaults to `pull`.
