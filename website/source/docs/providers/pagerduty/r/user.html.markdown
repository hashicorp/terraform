---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_user"
sidebar_current: "docs-pagerduty-resource-user"
description: |-
  Creates and manages a user in PagerDuty.
---

# pagerduty\_user

A [user](https://v2.developer.pagerduty.com/v2/page/api-reference#!/Users/get_users) is a member of a PagerDuty account that have the ability to interact with incidents and other data on the account.


## Example Usage

```hcl
resource "pagerduty_team" "example" {
  name        = "Engineering"
  description = "All engineering"
}

resource "pagerduty_user" "example" {
  name  = "Earline Greenholt"
  email = "125.greenholt.earline@graham.name"
  teams = ["${pagerduty_team.example.id}"]
}
```

## Argument Reference

The following arguments are supported:

  * `name` - (Required) The name of the user.
  * `email` - (Required) The user's email address.
  * `color` - (Optional) The schedule color for the user.
  * `role` - (Optional) The user role. Account must have the `read_only_users` ability to set a user as a `read_only_user`. Can be `admin`, `limited_user`, `owner`, `read_only_user`, `team_responder` or `user`
  * `job_title` - (Optional) The user's title.
  * `teams` - (Optional) A list of teams the user should belong to.
  * `description` - (Optional) A human-friendly description of the user.
    If not set, a placeholder of "Managed by Terraform" will be set.

## Attributes Reference

The following attributes are exported:

  * `id` - The ID of the user.
  * `avatar_url` - The URL of the user's avatar.
  * `html_url` - URL at which the entity is uniquely displayed in the Web app
  * `invitation_sent` - If true, the user has an outstanding invitation.

## Import

Users can be imported using the `id`, e.g.

```
$ terraform import pagerduty_user.main PLBP09X
```
