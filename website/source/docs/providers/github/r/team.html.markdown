---
layout: "github"
page_title: "Github: github_team"
sidebar_current: "docs-github-resource-team"
description: |-
  Provides a Github team resource.
---

# github\_team

Provides a Github team resource.

This resource allows you to add/remove teams from your organization. When applied,
a new team will be created. When destroyed, that team will be removed.

## Example Usage

```
# Add a team to the organization
resource "github_team" "some_team" {
	name = "some-team"
	description = "Some cool team"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the team.
* `description` - (Optional) A description of the team.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the created team.
