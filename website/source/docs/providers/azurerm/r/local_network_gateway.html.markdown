---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_local_network_gateway"
sidebar_current: "docs-azurerm-resource-local-network-gateway"
description: |-
  Creates a new local network gateway connection over which specific connections can be configured.
---

# azurerm\_local\_network\_gateway

Creates a new local network gateway connection over which specific connections can be configured.

## Example Usage

```hcl
resource "azurerm_local_network_gateway" "home" {
  name                = "backHome"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  gateway_address     = "12.13.14.15"
  address_space       = ["10.0.0.0/16"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the local network gateway. Changing this
    forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the local network gateway.

* `location` - (Required) The location/region where the local network gatway is
    created. Changing this forces a new resource to be created.

* `gateway_address` - (Required) The IP address of the gateway to which to
    connect.

* `address_space` - (Required) The list of string CIDRs representing the
    address spaces the gateway exposes.

## Attributes Reference

The following attributes are exported:

* `id` - The local network gateway unique ID within Azure.

## Import

Local Network Gateways can be imported using the `resource id`, e.g.

```
terraform import azurerm_local_network_gateway.lng1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/localNetworkGateways/lng1
```
