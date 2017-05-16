---
layout: "ignition"
page_title: "Ignition: ignition_user"
sidebar_current: "docs-ignition-datasource-user"
description: |-
  Describes the desired user additions to the passwd database.
---

# ignition\_user

Describes the desired user additions to the passwd database.

## Example Usage

```hcl
data "ignition_user" "foo" {
	name = "foo"
	home_dir = "/home/foo/"
	shell = "/bin/bash"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The username for the account.

* `password_hash` - (Optional) The encrypted password for the account.

* `ssh_authorized_keys` - (Optional) A list of SSH keys to be added to the user’s authorized_keys.

* `uid` - (Optional) The user ID of the new account.

* `gecos` - (Optional) The GECOS field of the new account.

* `home_dir` - (Optional) The home directory of the new account.

* `no_create_home` - (Optional) Whether or not to create the user’s home directory.

* `primary_group` - (Optional) The name or ID of the primary group of the new account.

* `groups` - (Optional) The list of supplementary groups of the new account.

* `no_user_group` - (Optional) Whether or not to create a group with the same name as the user.

* `no_log_init` - (Optional) Whether or not to add the user to the lastlog and faillog databases.

* `shell` - (Optional) The login shell of the new account.	

## Attributes Reference

The following attributes are exported:

* `id` - ID used to reference this resource in _ignition_config_.