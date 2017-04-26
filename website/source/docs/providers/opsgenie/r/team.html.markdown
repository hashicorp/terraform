---
layout: "opsgenie"
page_title: "OpsGenie: opsgenie_team"
sidebar_current: "docs-opsgenie-resource-team"
description: |-
  Manages a Team within OpsGenie.
---

# opsgenie\_team

Manages a Team within OpsGenie.

## Example Usage

```hcl
resource "opsgenie_user" "first" {
  username  = "user@domain.com"
  full_name = "Cookie Monster"
  role      = "User"
}

resource "opsgenie_user" "second" {
  username  = "eggman@dr-robotnik.com"
  full_name = "Dr Ivo Eggman Robotnik"
  role      = "User"
}

resource "opsgenie_team" "test" {
  name        = "example"
  description = "This team deals with all the things"

  member {
    username = "${opsgenie_user.first.username}"
    role     = "admin"
  }

  member {
    username = "${opsgenie_user.second.username}"
    role     = "user"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name associated with this team. OpsGenie defines that this must not be longer than 100 characters.

* `description` - (Optional) A description for this team.

* `member` - (Optional) A Member block as documented below.

`member` supports the following:

* `username` - (Required) The username for the member to add to this Team.
* `role` - (Required) The role for the user within the Team - can be either 'Admin' or 'User'.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the OpsGenie User.

## Import

Users can be imported using the `id`, e.g.

```
$ terraform import opsgenie_team.team1 812be1a1-32c8-4666-a7fb-03ecc385106c
```
