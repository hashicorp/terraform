---
layout: "ignition"
page_title: "Ignition: ignition_filesystem"
sidebar_current: "docs-ignition-datasource-filesystem"
description: |-
  Describes the desired state of a system’s filesystem.
---

# ignition\_filesystem

Describes the desired state of a the system’s filesystems to be configured and/or used with the _ignition\_file_ resource.

## Example Usage

```hcl
data "ignition_filesystem" "foo" {
	name = "root"
	mount {
		device = "/dev/disk/by-label/ROOT"
		format = "xfs"
		create = true
		options = ["-L", "ROOT"]
	}
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The identifier for the filesystem, internal to Ignition. This is only required if the filesystem needs to be referenced in the a _ignition\_files_ resource.

* `mount` - (Optional) Contains the set of mount and formatting options for the filesystem. A non-null entry indicates that the filesystem should be mounted before it is used by Ignition.

* `path` - (Optional) The mount-point of the filesystem. A non-null entry indicates that the filesystem has already been mounted by the system at the specified path. This is really only useful for _/sysroot_.


The `mount` block supports:

* `device` - (Required) The absolute path to the device. Devices are typically referenced by the _/dev/disk/by-*_ symlinks.

* `format` - (Required) The filesystem format (ext4, btrfs, or xfs).

* `create` - (Optional) Indicates if the filesystem shall be created.

* `force` - (Optional) Whether or not the create operation shall overwrite an existing filesystem. Only allowed if the filesystem is being created.

* `options` - (Optional) Any additional options to be passed to the format-specific mkfs utility. Only allowed if the filesystem is being created

## Attributes Reference

The following attributes are exported:

* `id` - ID used to reference this resource in _ignition_config_.