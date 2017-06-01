---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_container_registry"
sidebar_current: "docs-azurerm-resource-container-registry"
description: |-
  Create as an Azure Container Registry instance.
---

# azurerm\_container\_registry

Create as an Azure Container Registry instance.

~> **Note:** All arguments including the access key will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "resourceGroup1"
  location = "West US"
}

resource "azurerm_storage_account" "test" {
  name                = "storageAccount1"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  account_type        = "Standard_GRS"
}

resource "azurerm_container_registry" "test" {
  name                = "containerRegistry1"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  admin_enabled       = true
  sku                 = "Basic"

  storage_account {
    name       = "${azurerm_storage_account.test.name}"
    access_key = "${azurerm_storage_account.test.primary_access_key}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the Container Registry. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the Container Registry.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `admin_enabled` - (Optional) Specifies whether the admin user is enabled. Defaults to `false`.

* `storage_account` - (Required) A Storage Account block as documented below - which must be located in the same data center as the Container Registry.

* `sku` - (Optional) The SKU name of the the container registry. `Basic` is the only acceptable value at this time.

* `tags` - (Optional) A mapping of tags to assign to the resource.

`storage_account` supports the following:

* `name` - (Required) The name of the storage account, which must be in the same physical location as the Container Registry.
* `access_key` - (Required) The access key to the storage account.

## Attributes Reference

The following attributes are exported:

* `id` - The Container Registry ID.

* `login_server` - The URL that can be used to log into the container registry.

* `admin_username` - The Username associated with the Container Registry Admin account - if the admin account is enabled.

* `admin_password` - The Password associated with the Container Registry Admin account - if the admin account is enabled.

## Import

Container Registries can be imported using the `resource id`, e.g.

```
terraform import azurerm_container_registry.test /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/mygroup1/providers/Microsoft.ContainerRegistry/registries/myregistry1
```
