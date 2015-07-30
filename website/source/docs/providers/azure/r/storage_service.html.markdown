---
layout: "azure"
page_title: "Azure: azure_storage_service"
sidebar_current: "docs-azure-storage-service"
description: |-
    Creates a new storage service on Azure in which storage containers may be created.
---

# azure\_storage\_service

Creates a new storage service on Azure in which storage containers may be created.

## Example Usage

```
resource "azure_storage_service" "tfstor" {
    name = "tfstor"
    location = "West US"
    description = "Made by Terraform."
    account_type = "Standard_LRS"
}
````

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage service. Must be between 4 and 24
    lowercase-only characters or digits. Must be unique on Azure.

* `location` - (Required) The location where the storage service should be created.
    For a list of all Azure locations, please consult [this link](http://azure.microsoft.com/en-us/regions/).

* `account_type` - (Required) The type of storage account to be created.
    Available options include `Standard_LRS`, `Standard_ZRS`, `Standard_GRS`,
    `Standard_RAGRS` and `Premium_LRS`. To learn more about the differences
    of each storage account type, please consult [this link](http://blogs.msdn.com/b/windowsazurestorage/archive/2013/12/11/introducing-read-access-geo-replicated-storage-ra-grs-for-windows-azure-storage.aspx).

* `affinity_group` - (Optional) The affinity group the storage service should
    belong to.

* `properties` - (Optional) Key-value definition of additional properties
    associated to the storage service. For additional information on what
    these properties do, please consult [this link](https://msdn.microsoft.com/en-us/library/azure/hh452235.aspx).

* `label` - (Optional) A label to be used for tracking purposes. Must be
    non-void. Defaults to `Made by Terraform.`.

* `description` - (Optional) A description for the storage service.

## Attributes Reference

The following attributes are exported:

* `id` - The storage service ID. Coincides with the given `name`.
