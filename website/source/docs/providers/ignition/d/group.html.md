---
layout: "ignition"
page_title: "Ignition: ignition_group"
sidebar_current: "docs-ignition-datasource-group"
description: |-
  Describes the desired group additions to the passwd database.
---

# ignition\_group

Describes the desired group additions to the passwd database.

## Example Usage

```hcl
data "ignition_group" "foo" {
	name = "foo"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The groupname for the account.

* `password_hash` - (Optional) The encrypted password for the account.

* `gid` - (Optional) The group ID of the new account.

## Attributes Reference

The following attributes are exported:

* `id` - ID used to reference this resource in _ignition_config_.