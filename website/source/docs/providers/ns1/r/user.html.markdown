---
layout: "ns1"
page_title: "NS1: ns1_user"
sidebar_current: "docs-ns1-resource-user"
description: |-
  Provides a NS1 User resource.
---

# ns1\_user

Provides a NS1 User resource. Creating a user sends an invitation email to the user's email address. This can be used to create, modify, and delete users.

## Example Usage

```hcl
resource "ns1_team" "example" {
  name = "Example team"

  permissions = {
    dns_view_zones       = false
    account_manage_users = false
  }
}

resource "ns1_user" "example" {
  name     = "Example User"
  username = "example_user"
  email    = "user@example.com"
  teams    = ["${ns1_team.example.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The free form name of the user.
* `username` - (Required) The users login name.
* `email` - (Required) The email address of the user.
* `notify` - (Required) The Whether or not to notify the user of specified events. Only `billing` is available currently.
* `teams` - (Required) The teams that the user belongs to.
* `permissions` - (Optional) The allowed permissions of the user. Permissions documented below.

Permissions (`permissions`) support the following:

* `dns_view_zones` - (Optional) Whether the user can view the accounts zones.
* `dns_manage_zones` - (Optional) Whether the user can modify the accounts zones.
* `dns_zones_allow_by_default` - (Optional) If true, enable the `dns_zones_allow` list, otherwise enable the `dns_zones_deny` list.
* `dns_zones_allow` - (Optional) List of zones that the user may access.
* `dns_zones_deny` - (Optional) List of zones that the user may not access.
* `data_push_to_datafeeds` - (Optional) Whether the user can publish to data feeds.
* `data_manage_datasources` - (Optional) Whether the user can modify data sources.
* `data_manage_datafeeds` - (Optional) Whether the user can modify data feeds.
* `account_manage_users` - (Optional) Whether the user can modify account users.
* `account_manage_payment_methods` - (Optional) Whether the user can modify account payment methods.
* `account_manage_plan` - (Optional) Whether the user can modify the account plan.
* `account_manage_teams` - (Optional) Whether the user can modify other teams in the account.
* `account_manage_apikeys` - (Optional) Whether the user can modify account apikeys.
* `account_manage_account_settings` - (Optional) Whether the user can modify account settings.
* `account_view_activity_log` - (Optional) Whether the user can view activity logs.
* `account_view_invoices` - (Optional) Whether the user can view invoices.
* `monitoring_manage_lists` - (Optional) Whether the user can modify notification lists.
* `monitoring_manage_jobs` - (Optional) Whether the user can modify monitoring jobs.
* `monitoring_view_jobs` - (Optional) Whether the user can view monitoring jobs.

