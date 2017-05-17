---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_search_service"
sidebar_current: "docs-azurerm-resource-search-service"
description: |-
  Manage a Search Service.
---

# azurerm\_search\_service

Allows you to manage an Azure Search Service

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acceptanceTestResourceGroup1"
  location = "West US"
}

resource "azurerm_search_service" "test" {
  name                = "acceptanceTestSearchService1"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "West US"
  sku                 = "standard"

  tags {
    environment = "staging"
    database    = "test"
  }
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Search Service.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the Search Service.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `sku` - (Required) Valid values are `free` and `standard`. `standard2` is also valid, but can only be used when it's enabled on the backend by Microsoft support. `free` provisions the service in shared clusters. `standard` provisions the service in dedicated clusters

* `replica_count` - (Optional) Default is 1. Valid values include 1 through 12. Valid only when `sku` is `standard`.

* `partition_count` - (Optional) Default is 1. Valid values include 1, 2, 3, 4, 6, or 12. Valid only when `sku` is `standard`.

* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The Search Service ID.
