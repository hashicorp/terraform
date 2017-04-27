---
layout: "azurerm"
page_title: "Azure Resource Manager: azure_virtual_network_peering"
sidebar_current: "docs-azurerm-resource-network-virtual-network-peering"
description: |-
  Creates a new virtual network peering which allows resources to access other
  resources in the linked virtual network.
---

# azurerm\_virtual\_network\_peering

Creates a new virtual network peering which allows resources to access other
resources in the linked virtual network.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "peeredvnets-rg"
  location = "West US"
}

resource "azurerm_virtual_network" "test1" {
  name                = "peternetwork1"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space       = ["10.0.1.0/24"]
  location            = "West US"
}

resource "azurerm_virtual_network" "test2" {
  name                = "peternetwork2"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space       = ["10.0.2.0/24"]
  location            = "West US"
}

resource "azurerm_virtual_network_peering" "test1" {
  name                      = "peer1to2"
  resource_group_name       = "${azurerm_resource_group.test.name}"
  virtual_network_name      = "${azurerm_virtual_network.test1.name}"
  remote_virtual_network_id = "${azurerm_virtual_network.test2.id}"
}

resource "azurerm_virtual_network_peering" "test2" {
  name                      = "peer2to1"
  resource_group_name       = "${azurerm_resource_group.test.name}"
  virtual_network_name      = "${azurerm_virtual_network.test2.name}"
  remote_virtual_network_id = "${azurerm_virtual_network.test1.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual network peering. Changing this
    forces a new resource to be created.

* `virtual_network_name` - (Required) The name of the virtual network. Changing
    this forces a new resource to be created.

* `remote_virtual_network_id` - (Required) The full Azure resource ID of the
    remote virtual network.  Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the virtual network. Changing this forces a new resource to be
    created.

* `allow_virtual_network_access` - (Optional) Controls if the VMs in the remote
    virtual network can access VMs in the local virtual network. Defaults to
    false.

* `allow_forwarded_traffic` - (Optional) Controls if forwarded traffic from  VMs
    in the remote virtual network is allowed. Defaults to false.

* `allow_gateway_transit` - (Optional) Controls gatewayLinks can be used in the
    remote virtual network’s link to the local virtual network.

* `use_remote_gateways` - (Optional) Controls if remote gateways can be used on
    the local virtual network. If the flag is set to true, and
    allowGatewayTransit on the remote peering is also true, virtual network will
    use gateways of remote virtual network for transit. Only one peering can
    have this flag set to true. This flag cannot be set if virtual network
    already has a gateway. Defaults to false.

## Attributes Reference

The following attributes are exported:

* `id` - The Virtual Network Peering resource ID.

## Note

Virtual Network peerings cannot be created, updated or deleted concurrently.

## Import

Virtual Network Peerings can be imported using the `resource id`, e.g.

```
terraform import azurerm_virtual_network_peering.testPeering /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/virtualNetworks/myvnet1/virtualNetworkPeerings/myvnet1peering
```