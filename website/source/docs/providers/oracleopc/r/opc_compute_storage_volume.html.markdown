---
layout: "oracle"
page_title: "Oracle: opc_compute_storage_volume"
sidebar_current: "docs-opc-resource-storage_volume"
description: |-
  Creates and manages a storage volume in an OPC identity domain.
---

# opc\_compute\_storage\_volume

The ``opc_compute_storage_volume`` resource creates and manages a storage volume in an OPC identity domain.

~> **Caution:** The ``opc_compute_storage_volume`` resource can completely delete your
storage volume just as easily as it can create it. To avoid costly accidents,
consider setting
[``prevent_destroy``](/docs/configuration/resources.html#prevent_destroy)
on your storage volume resources as an extra safety measure.

## Example Usage

```
resource "opc_compute_storage_volume" "test_volume" {
       	size = "3g"
       	description = "My storage volume"
       	name = "test_volume_a"
       	tags = ["xyzzy", "quux"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within this identity domain) name of the storage volume.

* `size` - (Required) The size of the storage instance.

* `description` - (Optional) A description of the storage volume.

* `tags` - (Optional) A list of tags to apply to the storage volume.

* `bootableImage` - (Optional) The name of the bootable image the storage volume is loaded with.

* `bootableImageVersion` - (Optional) The version of the bootable image specified in `bootableImage` to use.

* `snapshot` - (Optional) The snapshot to initialise the storage volume with. This has two nested properties: `name`,
for the name of the snapshot to use, and `account` for the name of the snapshot account to use.

* `snapshotId` - (Optional) The id of the snapshot to initialise the storage volume with.
