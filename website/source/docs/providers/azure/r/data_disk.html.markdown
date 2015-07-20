---
layout: "azure"
page_title: "Azure: azure_data_disk"
sidebar_current: "docs-azure-resource-data-disk"
description: |-
  Adds a data disk to a virtual machine. If the name of an existing disk is given, it will attach that disk. Otherwise it will create and attach a new empty disk.
---

# azure\_data\_disk

Adds a data disk to a virtual machine. If the name of an existing disk is given,
it will attach that disk. Otherwise it will create and attach a new empty disk.

## Example Usage

```
resource "azure_data_disk" "data" {
    lun = 0
    size = 10
    storage_service_name = "yourstorage"
    virtual_machine = "server1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of an existing registered disk to attach to the
    virtual machine. If left empty, a new empty disk will be created and
    attached instead. Changing this forces a new resource to be created.

* `label` - (Optional) The identifier of the data disk. Changing this forces a
    new resource to be created (defaults to "virtual_machine-lun")

* `lun` - (Required) The Logical Unit Number (LUN) for the disk. The LUN
    specifies the slot in which the data drive appears when mounted for usage
    by the virtual machine. Valid LUN values are 0 through 31.

* `size` - (Optional) The size, in GB, of an empty disk to be attached to the
    virtual machine. Required when creating a new disk, not used otherwise.

* `caching` - (Optional) The caching behavior of data disk. Valid options are:
    `None`, `ReadOnly` and `ReadWrite` (defaults `None`)

* `storage_service_name` - (Optional) The name of an existing storage account
    within the subscription which will be used to store the VHD of this disk.
    Required if no value is supplied for `media_link`. Changing this forces
    a new resource to be created.

* `media_link` - (Optional) The location of the blob in storage where the VHD
    of this disk will be created. The storage account where must be associated
    with the subscription. Changing this forces a new resource to be created.

* `source_media_link` - (Optional) The location of a blob in storage where a
    VHD file is located that is imported and registered as a disk. If a value
    is supplied, `media_link` will not be used.

* `virtual_machine` - (Required) The name of the virtual machine the disk will
    be attached to.

## Attributes Reference

The following attributes are exported:

* `id` - The security group ID.
* `name` - The name of the disk.
* `label` - The identifier for the disk.
* `media_link` - The location of the blob in storage where the VHD of this disk
    is created.
