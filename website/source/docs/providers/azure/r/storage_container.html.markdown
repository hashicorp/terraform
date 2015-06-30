---
layout: "azure"
page_title: "Azure: azure_storage_container"
sidebar_current: "docs-azure-storage-container"
description: |-
    Creates a new storage container within a given storage service on Azure.
---

# azure\_storage\_container

Creates a new storage container within a given storage service on Azure.

## Example Usage

```
resource "azure_storage_container" "stor-cont" {
    name = "terraform-storage-container"
    container_access_type = "blob"
    storage_service_name = "tfstorserv"
}
````

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage container. Must be unique within
    the storage service the container is located.

* `storage_service_name` - (Required) The name of the storage service within
    which the storage container should be created.

* `container_access_type` - (Required) The 'interface' for access the container
    provides. Can be either `blob`, `container` or ``.

* `properties` - (Optional) Key-value definition of additional properties
    associated to the storage service.

## Attributes Reference

The following attributes are exported:

* `id` - The storage container ID. Coincides with the given `name`.
