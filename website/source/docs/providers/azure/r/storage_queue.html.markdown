---
layout: "azure"
page_title: "Azure: azure_storage_queue"
sidebar_current: "docs-azure-storage-queue"
description: |-
    Creates a new storage queue within a given storage service on Azure.
---

# azure\_storage\_queue

Creates a new storage queue within a given storage service on Azure.

## Example Usage

```
resource "azure_storage_queue" "stor-queue" {
    name = "terraform-storage-queue"
    storage_service_name = "tfstorserv"
}
````

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage queue. Must be unique within
    the storage service the queue is located.

* `storage_service_name` - (Required) The name of the storage service within
    which the storage queue should be created.

## Attributes Reference

The following attributes are exported:

* `id` - The storage queue ID. Coincides with the given `name`.
