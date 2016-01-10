---
layout: "azurerm"
page_title: "Azure Resource Manager: azure_subnet"
sidebar_current: "docs-azurerm-resource-network-subnet"
description: |-
  Creates a new subnet. Subnets represent network segments within the IP space defined by the virtual network.
---

# azurerm\_subnet

Creates a new subnet. Subnets represent network segments within the IP space defined by the virtual network.

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_subnet" "test" {
    name = "testsubnet"
    resource_group_name = "${azurerm_resource_group.test.name}"
    virtual_network_name = "${azurerm_virtual_network.test.name}"
    address_prefix = "10.0.1.0/24"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual network. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the subnet.

* `virtual_network_name` - (Required) The name of the virtual network to which to attach the subnet.

* `address_prefix` - (Required) The address prefix to use for the subnet.

* `network_security_group_id` - (Optional) The ID of the Network Security Group to associate with
    the subnet.
    
* `route_table_id` - (Optional) The ID of the Route Table to associate with
    the subnet.

## Attributes Reference

The following attributes are exported:

* `id` - The subnet ID.
* `ip_configurations` - The collection of IP Configurations with IPs within this subnet.
