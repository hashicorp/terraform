---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_route"
sidebar_current: "docs-azurerm-resource-network-route"
description: |-
  Creates a new Route Resource
---

# azurerm\_route

Creates a new Route Resource

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_route_table" "test" {
    name = "acceptanceTestRouteTable1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_route" "test" {
    name = "acceptanceTestRoute1"
    resource_group_name = "${azurerm_resource_group.test.name}"
    route_table_name = "${azurerm_route_table.test.name}"

    address_prefix = "10.1.0.0/16"
    next_hop_type = "vnetlocal"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the route. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the route.
    
    
* `route_table_name` - (Required) The name of the route table to which to create the route
    
* `address_prefix` - (Required) The destination CIDR to which the route applies, such as 10.1.0.0/16

* `next_hop_type` - (Required) The type of Azure hop the packet should be sent to.
                               Possible values are VirtualNetworkGateway, VnetLocal, Internet, VirtualAppliance and None

* `next_hop_in_ip_address` - (Optional) Contains the IP address packets should be forwarded to. Next hop values are only allowed in routes where the next hop type is VirtualAppliance.

## Attributes Reference

The following attributes are exported:

* `id` - The Route ID.
