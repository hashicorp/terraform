---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_storage_account"
sidebar_current: "docs-azurerm-resource-storage-account"
description: |-
  Create a Azure Storage Account.
---

# azurerm\_storage\_account

Create an Azure Storage Account.

## Example Usage

```
resource "azurerm_resource_group" "testrg" {
    name = "resourceGroupName"
    location = "westus"
}

resource "azurerm_storage_account" "testsa" {
    name = "storageaccountname"
    resource_group_name = "${azurerm_resource_group.testrg.name}"

    location = "westus"
    account_type = "Standard_GRS"

    tags {
        environment = "staging"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the storage account. Changing this forces a
    new resource to be created. This must be unique across the entire Azure service,
    not just within the resource group.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the storage account. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the
    resource exists. Changing this forces a new resource to be created.

* `account_type` - (Required) Defines the type of storage account to be
    created. Valid options are `Standard_LRS`, `Standard_ZRS`, `Standard_GRS`,
    `Standard_RAGRS`, `Premium_LRS`. Changing this is sometimes valid - see the Azure
    documentation for more information on which types of accounts can be converted
    into other types.

* `tags` - (Optional) A mapping of tags to assign to the resource.

Note that although the Azure API supports setting custom domain names for
storage accounts, this is not currently supported.

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The storage account Resource ID.
* `primary_location` - The primary location of the storage account.
* `secondary_location` - The secondary location of the storage account.
* `primary_blob_endpoint` - The endpoint URL for blob storage in the primary location.
* `secondary_blob_endpoint` - The endpoint URL for blob storage in the secondary location.
* `primary_queue_endpoint` - The endpoint URL for queue storage in the primary location.
* `secondary_queue_endpoint` - The endpoint URL for queue storage in the secondary location.
* `primary_table_endpoint` - The endpoint URL for table storage in the primary location.
* `secondary_table_endpoint` - The endpoint URL for table storage in the secondary location.
* `primary_file_endpoint` - The endpoint URL for file storage in the primary location.
