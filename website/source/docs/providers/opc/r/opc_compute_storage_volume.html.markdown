---
layout: "opc"
page_title: "Oracle: opc_compute_storage_volume"
sidebar_current: "docs-opc-resource-storage-volume"
description: |-
  Creates and manages a storage volume in an OPC identity domain.
---

# opc\_compute\_storage\_volume

The ``opc_compute_storage_volume`` resource creates and manages a storage volume in an OPC identity domain.

~> **Caution:** The ``opc_compute_storage_volume`` resource can completely delete your storage volume just as easily as it can create it. To avoid costly accidents, consider setting [``prevent_destroy``](/docs/configuration/resources.html#prevent_destroy) on your storage volume resources as an extra safety measure.

## Example Usage

```
resource "opc_compute_storage_volume" "test" {
  name        = "storageVolume1"
  description = "Description for the Storage Volume"
  size        = 10
  tags        = ["bar", "foo"]
}
```

## Example Usage (Bootable Volume)
```
resource "opc_compute_image_list" "test" {
  name        = "imageList1"
  description = "Description for the Image List"
}

resource "opc_compute_storage_volume" "test" {
  name        = "storageVolume1"
  description = "Description for the Bootable Storage Volume"
  size        = 30
  tags        = ["first", "second"]
  bootable {
  	image_list = "${opc_compute_image_list.test.name}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` (Required) The name for the Storage Account.
* `description` (Optional) The description of the storage volume.
* `size` (Required) The size of this storage volume in GB. The allowed range is from 1 GB to 2 TB (2048 GB).
* `storage_type` - (Optional) - The Type of Storage to provision. Possible values are `/oracle/public/storage/latency` or `/oracle/public/storage/default`. Defaults to `/oracle/public/storage/default`.
* `bootable` - (Optional) A `bootable` block as defined below.
* `tags` - (Optional) Comma-separated strings that tag the storage volume.

`bootable` supports the following:
* `image_list` - (Optional) Defines an image list.
* `image_list_entry` - (Optional) Defines an image list entry.

## Attributes Reference

The following attributes are exported:

* `hypervisor` - The hypervisor that this volume is compatible with.
* `machine_image` - Name of the Machine Image - available if the volume is a bootable storage volume.
* `managed` - Is this a Managed Volume?
* `platform` - The OS platform this volume is compatible with.
* `readonly` - Can this Volume be attached as readonly?
* `status` - The current state of the storage volume.
* `storage_pool` - The storage pool from which this volume is allocated.
* `uri` - Unique Resource Identifier of the Storage Volume.

## Import

Storage Volume's can be imported using the `resource name`, e.g.

```
terraform import opc_compute_storage_volume.volume1 example
```
