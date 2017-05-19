---
layout: "ignition"
page_title: "Ignition: ignition_config"
sidebar_current: "docs-ignition-datasource-config"
description: |-
  Renders an ignition configuration as JSON
---

# ignition\_config

Renders an ignition configuration as JSON. It  contains all the disks, partitions, arrays, filesystems, files, users, groups and units.

## Example Usage

```hcl
data "ignition_config" "example" {
	systemd = [
		"${data.ignition_systemd_unit.example.id}",
	]
}
```

## Argument Reference

The following arguments are supported:

* `disks` - (Optional) The list of disks to be configured and their options.

* `arrays` - (Optional) The list of RAID arrays to be configured.

* `filesystems` - (Optional) The list of filesystems to be configured and/or used in the _ignition_file_ resource.

* `files` - (Optional) The list of files, rooted in this particular filesystem, to be written.

* `systemd` - (Optional) The list of systemd units. Describes the desired state of the systemd units.

* `networkd` - (Optional) The list of networkd units. Describes the desired state of the networkd files.

* `users` - (Optional) The list of accounts to be added.

* `groups` - (Optional) The list of groups to be added.

* `append` - (Optional) Any number of blocks with the configs to be appended to the current config.

* `replace` - (Optional) A block with config that will replace the current.


The `append` and `replace` blocks supports:

* `source` - (Required) The URL of the config. Supported schemes are http. Note: When using http, it is advisable to use the verification option to ensure the contents haven’t been modified.

* `verification` - (Optional) The hash of the config, in the form _\<type\>-\<value\>_ where type is sha512.

## Attributes Reference

The following attributes are exported:

* `rendered` - The final rendered template.
