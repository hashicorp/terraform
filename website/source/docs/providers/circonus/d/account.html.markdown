---
layout: "circonus"
page_title: "Circonus: account"
sidebar_current: "docs-circonus-datasource-account"
description: |-
    Provides details about a specific Circonus Account.
---

# circonus_account

`circonus_account` provides
[details](https://login.circonus.com/resources/api/calls/account) about a specific
[Circonus Account](https://login.circonus.com/user/docs/Administration/Account).

The `circonus_account` data source can be used for pulling various attributes
about a specific Circonus Account.

## Example Usage

The following example shows how the resource might be used to obtain the metrics
usage and limit of a given Circonus Account.

```hcl
data "circonus_account" "current" {
  current = true
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available
regions. The given filters must match exactly one region whose data will be
exported as attributes.

* `id` - (Optional) The Circonus ID of a given account.
* `current` - (Optional) Automatically use the current Circonus Account attached
  to the API token making the request.

At least one of the above attributes should be provided when searching for a
account.

## Attributes Reference

The following attributes are exported:

* `address1` - The first line of the address associated with the account.

* `address2` - The second line of the address associated with the account.

* `cc_email` - An optionally specified email address used in the CC line of invoices.

* `id` - The Circonus ID of the selected Account.

* `city` - The city part of the address associated with the account.

* `contact_groups` - A list of IDs for each contact group in the account.

* `country` - The country of the user's address.

* `description` - Description of the account.

* `invites` - An list of users invited to use the platform.  Each element in the
  list has both an `email` and `role` attribute.

* `name` - The name of the account.

* `owner` - The Circonus ID of the user who owns this account.

* `state_prov` - The state or province of the address associated with the account.

* `timezone` - The timezone that events will be displayed in the web interface
  for this account.

* `ui_base_url` - The base URL of this account.

* `usage` - A list of account usage limits.  Each element in the list will have
  a `limit` attribute, a limit `type`, and a `used` attribute.

* `users` - A list of users who have access to this account.  Each element in
  the list has both an `id` and a `role`.  The `id` is a Circonus ID referencing
  the user.
