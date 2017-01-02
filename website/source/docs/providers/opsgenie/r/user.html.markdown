---
layout: "opsgenie"
page_title: "OpsGenie: opsgenie_user"
sidebar_current: "docs-opsgenie-resource-user"
description: |-
  Manages a User within OpsGenie.
---

# opsgenie\_user

Manages a User within OpsGenie.

## Example Usage

```
resource "opsgenie_user" "test" {
  username  = "user@domain.com"
  full_name = "Cookie Monster"
  role      = "User"
}
```

## Argument Reference

The following arguments are supported:

* `username` - (Required) The email address associated with this user. OpsGenie defines that this must not be longer than 100 characters.

* `full_name` - (Required) The Full Name of the User.

* `role` - (Required) The Role assigned to the User. Either a built-in such as 'Owner', 'Admin' or 'User' - or the name of a custom role.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the OpsGenie User.

## Import

Users can be imported using the `id`, e.g.

```
$ terraform import opsgenie_user.user da4faf16-5546-41e4-8330-4d0002b74048
```
