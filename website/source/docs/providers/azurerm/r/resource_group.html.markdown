---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_resource_group"
sidebar_current: "docs-azurerm-resource-resource-group"
description: |-
    Creates a new resource group on Azure.
---

# azurerm\_resource\_group

Creates a new resource group on Azure.

## Example Usage

```
resource "azurerm_resource_group" "test" {
  name     = "testResourceGroup1"
  location = "West US"
  
  tags {
    environment = "Production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the resource group. Must be unique on your
    Azure subscription.

* `location` - (Required) The location where the resource group should be created.
    For a list of all Azure locations, please consult [this link](http://azure.microsoft.com/en-us/regions/).
    
* `tags` - (Optional) A mapping of tags to assign to the resource. 

## Attributes Reference

The following attributes are exported:

* `id` - The resource group ID.
