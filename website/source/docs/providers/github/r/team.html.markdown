---
layout: "github"
page_title: "GitHub: github_team"
sidebar_current: "docs-github-resource-team"
description: |-
  Provides a GitHub team resource.
---

# github\_team

Provides a GitHub team resource.

This resource allows you to add/remove teams from your organization. When applied,
a new team will be created. When destroyed, that team will be removed.

## Example Usage

```
# Add a team to the organization
resource "github_team" "some_team" {
	name = "some-team"
	description = "Some cool team"
	privacy = "closed"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the team.
* `description` - (Optional) A description of the team.
* `privacy` - (Optional) The level of privacy for the team. Must be one of `secret` or `closed`.
               Defaults to `secret`.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the created team.
