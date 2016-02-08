---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_storage_blob"
sidebar_current: "docs-azurerm-resource-storage-blob"
description: |-
  Create a Azure Storage Blob.
---

# azurerm\_storage\_blob

Create an Azure Storage Blob.

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "westus"
}

resource "azurerm_storage_account" "test" {
    name = "acctestacc%s"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "westus"
    account_type = "Standard_LRS"
}

resource "azurerm_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    container_access_type = "private"
}

resource "azurerm_storage_blob" "testsb" {
    name = "sample.vhd"

    resource_group_name = "${azurerm_resource_group.test.name}"
    storage_account_name = "${azurerm_storage_account.test.name}"
    storage_container_name = "${azurerm_storage_container.test.name}"

    type = "page"
    size = 5120
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage blob. Must be unique within the storage container the blob is located.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the storage container. Changing this forces a new resource to be created.

* `storage_account_name` - (Required) Specifies the storage account in which to create the storage container.
 Changing this forces a new resource to be created.

* `storage_container_name` - (Required) The name of the storage container in which this blob should be created.

* `type` - (Required) The type of the storage blob to be created. One of either `block` or `page`.

* `size` - (Optional) Used only for `page` blobs to specify the size in bytes of the blob to be created. Must be a multiple of 512. Defaults to 0. 

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The storage blob Resource ID.
* `url` - The URL of the blob
