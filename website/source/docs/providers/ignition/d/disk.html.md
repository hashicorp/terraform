---
layout: "ignition"
page_title: "Ignition: ignition_disk"
sidebar_current: "docs-ignition-datasource-disk"
description: |-
  Describes the desired state of a system’s disk.
---

# ignition\_disk

Describes the desired state of a system’s disk.

## Example Usage

```hcl
data "ignition_disk" "foo" {
	device = "/dev/sda"
	partition {
		start = 2048
		size = 196037632
	}
}
```

## Argument Reference

The following arguments are supported:

* `device` - (Required) The absolute path to the device. Devices are typically referenced by the _/dev/disk/by-*_ symlinks.

* `wipe_table` - (Optional) Whether or not the partition tables shall be wiped. When true, the partition tables are erased before any further manipulation. Otherwise, the existing entries are left intact.

* `partition` - (Optional) The list of partitions and their configuration for this particular disk..


The `partition` block supports:
 
* `label` - (Optional) The PARTLABEL for the partition.

* `number` - (Optional) The partition number, which dictates it’s position in the partition table (one-indexed). If zero, use the next available partition slot.

* `size` - (Optional) The size of the partition (in sectors). If zero, the partition will fill the remainder of the disk.


* `start` - (Optional) The start of the partition (in sectors). If zero, the partition will be positioned at the earliest available part of the disk.


* `type_guid` - (Optional) The GPT [partition type GUID](http://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_type_GUIDs). If omitted, the default will be _0FC63DAF-8483-4772-8E79-3D69D8477DE4_ (Linux filesystem data).

## Attributes Reference

The following attributes are exported:

* `id` - ID used to reference this resource in _ignition_config_.