---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_storage_share"
sidebar_current: "docs-azurerm-resource-storage-share"
description: |-
  Create an Azure Storage Share.
---

# azurerm\_storage\_share

Create an Azure Storage File Share.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = "westus"
}

resource "azurerm_storage_account" "test" {
  name                = "acctestacc%s"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "westus"
  account_type        = "Standard_LRS"
}

resource "azurerm_storage_share" "testshare" {
  name = "sharename"

  resource_group_name  = "${azurerm_resource_group.test.name}"
  storage_account_name = "${azurerm_storage_account.test.name}"

  quota = 50
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the share. Must be unique within the storage account where the share is located.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the share. Changing this forces a new resource to be created.

* `storage_account_name` - (Required) Specifies the storage account in which to create the share.
 Changing this forces a new resource to be created.

* `quota` - (Optional) The maximum size of the share, in gigabytes. Must be greater than 0, and less than or equal to 5 TB (5120 GB). Default this is set to 0 which results in setting the quota to 5 TB.


## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The storage share Resource ID.
* `url` - The URL of the share
