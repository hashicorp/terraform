---
layout: "oracleopc"
page_title: "Oracle: opc_compute_instance"
sidebar_current: "docs-oracleopc-resource-instance"
description: |-
  Creates and manages an instance in an OPC identity domain.
---

# opc\_compute\_instance

The ``opc_compute_instance`` resource creates and manages an instance in an OPC identity domain.

~> **Caution:** The ``opc_compute_instance`` resource can completely delete your
instance just as easily as it can create it. To avoid costly accidents,
consider setting
[``prevent_destroy``](/docs/configuration/resources.html#prevent_destroy)
on your instance resources as an extra safety measure.

## Example Usage

```
resource "opc_compute_instance" "test_instance" {
       	name = "test"
       	label = "test"
       	shape = "oc3"
       	imageList = "/oracle/public/oel_6.4_2GB_v1"
       	sshKeys = ["${opc_compute_ssh_key.key1.name}"]
       	attributes = "{\"foo\":\"bar\"}"
       	storage = [{
       		index = 1
       		volume = "${opc_compute_storage_volume.test_volume.name}"
       	},
       	{
       		index = 2
       		volume = "${opc_compute_storage_volume.test_volume2.name}"
       	}]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the instance. This need not be unique, as each instance is assigned a separate
computed `opcId`.

* `shape` - (Required) The shape of the instance, e.g. `oc4`.

* `imageList` - (Optional) The imageList of the instance, e.g. `/oracle/public/oel_6.4_2GB_v1`

* `label` - (Optional) The label to apply to the instance.

* `ip` - (Computed) The internal IP address assigned to the instance.

* `opcId` - (Computed) The interned ID assigned to the instance.

* `sshKeys` - (Optional) The names of the SSH Keys that can be used to log into the instance.

* `attributes` - (Optional) An arbitrary JSON-formatted collection of attributes which is made available to the instance.

* `vcable` - (Computed) The ID of the instance's VCable, which is used to associate it with reserved IP addresses and
add it to Security Lists.

* `storage` - (Optional) A set of zero or more storage volumes to attach to the instance. Each volume has two arguments:
`index`, which is the volume's index in the instance's list of mounted volumes, and `name`, which is the name of the
storage volume to mount.

* `bootOrder` - (Optional) The index number of the bootable storage volume that should be used to boot the instance. e.g. `[ 1 ]`.  If you specify both `bootOrder` and `imageList`, the imagelist attribute is ignored.
