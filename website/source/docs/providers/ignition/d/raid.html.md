---
layout: "ignition"
page_title: "Ignition: ignition_raid"
sidebar_current: "docs-ignition-datasource-raid"
description: |-
  Describes the desired state of the system’s RAID.
---

# ignition\_raid

Describes the desired state of the system’s RAID.

## Example Usage

```hcl
data "ignition_raid" "md" {
	name = "data"
	level = "stripe"
	devices = [
      	"/dev/disk/by-partlabel/raid.1.1",
        "/dev/disk/by-partlabel/raid.1.2"
	]
}

data "ignition_disk" "disk1" {
	device = "/dev/sdb"
	wipe_table = true
	partition {
		label = "raid.1.1"
        number = 1
        size = 20480
        start = 0
	}
}

data "ignition_disk" "disk2" {
	device = "/dev/sdc"
	wipe_table = true
	partition {
		label = "raid.1.2"
        number = 1
        size = 20480
        start = 0
	}
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name to use for the resulting md device.

* `level` - (Required) The redundancy level of the array (e.g. linear, raid1, raid5, etc.).

* `devices` - (Required) The list of devices (referenced by their absolute path) in the array.

* `spares` - (Optional) The number of spares (if applicable) in the array.

## Attributes Reference

The following attributes are exported:

* `id` - ID used to reference this resource in _ignition_config_