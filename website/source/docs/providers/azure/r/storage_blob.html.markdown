---
layout: "azure"
page_title: "Azure: azure_storage_blob"
sidebar_current: "docs-azure-storage-blob"
description: |-
    Creates a new storage blob within a given storage container on Azure.
---

# azure\_storage\_blob

Creates a new storage blob within a given storage container on Azure.

## Example Usage

```
resource "azure_storage_blob" "foo" {
    name = "tftesting-blob"
    storage_service_name = "tfstorserv"
    storage_container_name = "terraform-storage-container"
    type = "PageBlob"
    size = 1024
}
````

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage blob. Must be unique within
    the storage service the blob is located.

* `storage_service_name` - (Required) The name of the storage service within
    which the storage container in which the blob will be created resides.

* `storage_container_name` - (Required) The name of the storage container
    in which this blob should be created. Must be located on the storage
    service given with `storage_service_name`.

* `type` - (Required) The type of the storage blob to be created. One of either
    `BlockBlob` or `PageBlob`.

* `size` - (Optional) Used only for `PageBlob`'s to specify the size in bytes
    of the blob to be created. Must be a multiple of 512. Defaults to 0.

## Attributes Reference

The following attributes are exported:

* `id` - The storage blob ID. Coincides with the given `name`.
