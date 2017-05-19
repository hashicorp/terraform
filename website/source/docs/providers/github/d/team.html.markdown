---
layout: "github"
page_title: "Github: github_team"
sidebar_current: "docs-github-datasource-team"
description: |-
  Get information on a Github team.
---

# github\_team

Use this data source to retrieve information about a Github team.

## Example Usage

```
data "github_team" "example" {
  slug = "example"
}
```

## Argument Reference

 * `slug` - (Required) The team slug.

## Attributes Reference

 * `name` - the team's full name.
 * `description` - the team's description.
 * `privacy` - the team's privacy type.
 * `permission` - the team's permission level.
