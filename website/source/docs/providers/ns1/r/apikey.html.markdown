---
layout: "ns1"
page_title: "NS1: ns1_apikey"
sidebar_current: "docs-ns1-resource-apikey"
description: |-
  Provides a NS1 Api Key resource.
---

# ns1\_apikey

Provides a NS1 Api Key resource. This can be used to create, modify, and delete api keys.

## Example Usage

```hcl
resource "ns1_team" "example" {
  name = "Example team"
}

resource "ns1_apikey" "example" {
  name  = "Example key"
  teams = ["${ns1_team.example.id}"]

  permissions = {
    dns_view_zones       = false
    account_manage_users = false
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The free form name of the apikey.
* `key` - (Required) The apikeys authentication token.
* `teams` - (Required) The teams that the apikey belongs to.
* `permissions` - (Optional) The allowed permissions of the apikey. Permissions documented below.

Permissions (`permissions`) support the following:

* `dns_view_zones` - (Optional) Whether the apikey can view the accounts zones.
* `dns_manage_zones` - (Optional) Whether the apikey can modify the accounts zones.
* `dns_zones_allow_by_default` - (Optional) If true, enable the `dns_zones_allow` list, otherwise enable the `dns_zones_deny` list.
* `dns_zones_allow` - (Optional) List of zones that the apikey may access.
* `dns_zones_deny` - (Optional) List of zones that the apikey may not access.
* `data_push_to_datafeeds` - (Optional) Whether the apikey can publish to data feeds.
* `data_manage_datasources` - (Optional) Whether the apikey can modify data sources.
* `data_manage_datafeeds` - (Optional) Whether the apikey can modify data feeds.
* `account_manage_users` - (Optional) Whether the apikey can modify account users.
* `account_manage_payment_methods` - (Optional) Whether the apikey can modify account payment methods.
* `account_manage_plan` - (Optional) Whether the apikey can modify the account plan.
* `account_manage_teams` - (Optional) Whether the apikey can modify other teams in the account.
* `account_manage_apikeys` - (Optional) Whether the apikey can modify account apikeys.
* `account_manage_account_settings` - (Optional) Whether the apikey can modify account settings.
* `account_view_activity_log` - (Optional) Whether the apikey can view activity logs.
* `account_view_invoices` - (Optional) Whether the apikey can view invoices.
* `monitoring_manage_lists` - (Optional) Whether the apikey can modify notification lists.
* `monitoring_manage_jobs` - (Optional) Whether the apikey can modify monitoring jobs.
* `monitoring_view_jobs` - (Optional) Whether the apikey can view monitoring jobs.

