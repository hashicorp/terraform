---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_route_table"
sidebar_current: "docs-azurerm-resource-network-route-table"
description: |-
  Creates a new Route Table Resource
---

# azurerm\_route\_table

Creates a new Route Table Resource

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acceptanceTestResourceGroup1"
  location = "West US"
}

resource "azurerm_route_table" "test" {
  name                = "acceptanceTestSecurityGroup1"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  route {
    name           = "route1"
    address_prefix = "10.1.0.0/16"
    next_hop_type  = "vnetlocal"
  }

  tags {
    environment = "Production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the route table. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the route table.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `route` - (Optional) Can be specified multiple times to define multiple
                                   routes. Each `route` block supports fields documented below.

* `tags` - (Optional) A mapping of tags to assign to the resource.

The `route` block supports:

* `name` - (Required) The name of the route.

* `address_prefix` - (Required) The destination CIDR to which the route applies, such as 10.1.0.0/16

* `next_hop_type` - (Required) The type of Azure hop the packet should be sent to.
                               Possible values are VirtualNetworkGateway, VnetLocal, Internet, VirtualAppliance and None

* `next_hop_in_ip_address` - (Optional) Contains the IP address packets should be forwarded to. Next hop values are only allowed in routes where the next hop type is VirtualAppliance.

## Attributes Reference

The following attributes are exported:

* `id` - The Route Table ID.
* `subnets` - The collection of Subnets associated with this route table.

## Import


Route Tables can be imported using the `resource id`, e.g.
```
terraform import azurerm_route_table.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/routeTables/mytable1
```