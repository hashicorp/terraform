---
layout: "azurerm"
page_title: "Azure Resource Manager: azure_virtual_network_gateway_connection"
sidebar_current: "docs-azurerm-resource-network-virtual-network-gateway-connection"
description: |-
  Creates a new connection in an existing virtual network gateway.
---

# azurerm\_virtual\_network\_gateway\_connection

Creates a new connection in an existing virtual network gateway.

## Example Usage

### Site-to-Site connection

The following example shows a connection between an Azure virtual network
and an on-premises VPN device and network.

```
resource "azurerm_resource_group" "test" {
    name = "test"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
  name = "test"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space = ["10.0.0.0/16"]
}

resource "azurerm_subnet" "test" {
  name = "GatewaySubnet"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix = "10.0.1.0/24"
}

resource "azurerm_local_network_gateway" "onpremise" {
  name = "onpremise"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  gateway_address = "168.62.225.23"
  address_space = ["10.1.1.0/24"]
}

resource "azurerm_public_ip" "test" {
  name = "test"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "test" {
  name = "test"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  type = "Vpn"
  vpn_type = "RouteBased"

  active_active = false
  enable_bgp = false

	sku {
		name = "Basic"
		tier = "Basic"
	}

  ip_configuration {
    name = "vnetGatewayConfig"
    public_ip_address_id = "${azurerm_public_ip.test.id}"
    private_ip_address_allocation = "Dynamic"
    subnet_id = "${azurerm_subnet.test.id}"
  }
}

resource "azurerm_virtual_network_gateway_connection" "onpremise" {
  name = "onpremise"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  type = "IPsec"
  virtual_network_gateway_id = "${azurerm_virtual_network_gateway.test.id}"
  local_network_gateway_id = "${azurerm_local_network_gateway.onpremise.id}"

  shared_key = "4-v3ry-53cr37-1p53c-5h4r3d-k3y"
}
```

### VNet-to-VNet connection

The following example shows a connection between two Azure virtual network
in different locations/regions.

```
resource "azurerm_resource_group" "us" {
    name = "us"
    location = "East US"
}

resource "azurerm_virtual_network" "us" {
  name = "us"
  location = "East US"
  resource_group_name = "${azurerm_resource_group.us.name}"
  address_space = ["10.0.0.0/16"]
}

resource "azurerm_subnet" "us_gateway" {
  name = "GatewaySubnet"
  resource_group_name = "${azurerm_resource_group.us.name}"
  virtual_network_name = "${azurerm_virtual_network.us.name}"
  address_prefix = "10.0.1.0/24"
}

resource "azurerm_public_ip" "us" {
  name = "us"
  location = "East US"
  resource_group_name = "${azurerm_resource_group.us.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "us" {
  name = "us-gateway"
  location = "East US"
  resource_group_name = "${azurerm_resource_group.us.name}"

  type = "Vpn"
  vpn_type = "RouteBased"

	sku {
		name = "Basic"
		tier = "Basic"
	}

  ip_configuration {
    name = "vnetGatewayConfig"
    public_ip_address_id = "${azurerm_public_ip.us.id}"
    private_ip_address_allocation = "Dynamic"
    subnet_id = "${azurerm_subnet.us_gateway.id}"
  }
}

resource "azurerm_resource_group" "europe" {
  name = "europe"
  location = "West Europe"
}

resource "azurerm_virtual_network" "europe" {
  name = "europe"
  location = "West Europe"
  resource_group_name = "${azurerm_resource_group.europe.name}"
  address_space = ["10.1.0.0/16"]
}

resource "azurerm_subnet" "europe_gateway" {
  name = "GatewaySubnet"
  resource_group_name = "${azurerm_resource_group.europe.name}"
  virtual_network_name = "${azurerm_virtual_network.europe.name}"
  address_prefix = "10.1.1.0/24"
}

resource "azurerm_public_ip" "europe" {
  name = "europe"
  location = "West Europe"
  resource_group_name = "${azurerm_resource_group.europe.name}"
  public_ip_address_allocation = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "europe" {
  name = "europe-gateway"
  location = "West Europe"
  resource_group_name = "${azurerm_resource_group.europe.name}"

  type = "Vpn"
  vpn_type = "RouteBased"

	sku {
		name = "Basic"
		tier = "Basic"
	}

  ip_configuration {
    name = "vnetGatewayConfig"
    public_ip_address_id = "${azurerm_public_ip.europe.id}"
    private_ip_address_allocation = "Dynamic"
    subnet_id = "${azurerm_subnet.europe_gateway.id}"
  }
}

resource "azurerm_virtual_network_gateway_connection" "us_to_europe" {
  name = "us-to-europe"
  location = "East US"
  resource_group_name = "${azurerm_resource_group.europe.name}"

  type = "Vnet2Vnet"
  virtual_network_gateway_id = "${azurerm_virtual_network_gateway.us.id}"
  peer_virtual_network_gateway_id = "${azurerm_virtual_network_gateway.europe.id}"

  shared_key = "4-v3ry-53cr37-1p53c-5h4r3d-k3y"
}

resource "azurerm_virtual_network_gateway_connection" "europe_to_us" {
  name = "europe-to-us"
  location = "West Europe"
  resource_group_name = "${azurerm_resource_group.europe.name}"

  type = "Vnet2Vnet"
  virtual_network_gateway_id = "${azurerm_virtual_network_gateway.europe.id}"
  peer_virtual_network_gateway_id = "${azurerm_virtual_network_gateway.us.id}"

  shared_key = "4-v3ry-53cr37-1p53c-5h4r3d-k3y"
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the connection. Changing the name forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the connection.

* `location` - (Required) The location/region where the connection is
    created. Changing this forces a new resource to be created.

* `type` - (Required) The type of connection. Valid options are `IPsec`
    (Site-to-Site), `ExpressRoute` (ExpressRoute), and `Vnet2Vnet` (VNet-to-VNet).
    Each connection type requires different mandatory arguments (refer to the
    examples above). Changing the connection type will force a new connection
    to be created.

* `virtual_network_gateway_id` - (Required) The full Azure resource ID of the
    virtual network gateway in which the connection will be created. Changing
    the gateway forces a new resource to be created.

* `authorization_key` - (Optional) The authorization key is required when
    creating an ExpressRoute connection to an Express Route Circuit which is
    contained in a different Azure subscription. This key is created by the owner
    of the Express Route Circuit to connect to.

* `express_route_circuit_id` - (Optional) The full Azure resource ID of the
    Express Route Circuit when creating an ExpressRoute connection. The
    Express Route Circuit can be in the same or in a different subscription.

* `peer_virtual_network_gateway_id` - (Optional) The full Azure resource ID
    of the peer virtual network gateway when creating a VNet-to-VNet connection.
    The peer virtual network gateway can be in the same or in a different subscription.

* `local_network_gateway_id` - (Optional) The full Azure resource ID of the
    local network gateway when creating Site-to-Site connection.

* `routing_weight` - (Optional) The routing weight. The default value is 10.

* `shared_key` - (Optional) The shared IPSec key.

* `enable_bgp` - (Optional) If true, BGP (Border Gateway Protocol) is enabled
    for this connection. By default, BGP is disabled.

* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The connection ID.

* `name` - The name of the connection.

* `resource_group_name` - The name of the resource group in which to create the virtual network.

* `location` - The location/region where the virtual network is created.

* `connection_status` - The current status of the connection.
    (`Connected`, `Connecting`, `NotConnected`, `Unknown`)

* `egress_bytes_transferred` - The egress bytes transferred in this connection.

* `ingress_bytes_transferred` - The ingress bytes transferred in this connection.

## Import

Virtual Network Gateway Connections can be imported using their `resource id`, e.g.

```
terraform import azurerm_virtual_network_gateway_connection.testConnection /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myGroup1/providers/Microsoft.Network/connections/myConnection1
```
