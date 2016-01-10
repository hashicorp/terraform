---
layout: "azurerm"
page_title: "Azure Resource Manager: azure_virtual_network"
sidebar_current: "docs-azurerm-resource-virtual-network"
description: |-
  Creates a new virtual network including any configured subnets. Each subnet can optionally be configured with a security group to be associated with the subnet.
---

# azurerm\_virtual\_network

Creates a new virtual network including any configured subnets. Each subnet can
optionally be configured with a security group to be associated with the subnet.

## Example Usage

```
resource "azurerm_virtual_network" "test" {
  name                = "virtualNetwork1"
  resource_group_name = "${azurerm_resource_group.test.name}"
  address_space       = ["10.0.0.0/16"]
  location            = "West US"

  subnet {
    name           = "subnet1"
    address_prefix = "10.0.1.0/24"
  }

  subnet {
    name           = "subnet2"
    address_prefix = "10.0.2.0/24"
  }

  subnet {
    name           = "subnet3"
    address_prefix = "10.0.3.0/24"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the network interface. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the network interface.

* `location` - (Required) The location/region where the network interface is
    created. Changing this forces a new resource to be created.

* `network_security_group_id` - (Optional) The ID of the Network Security Group to associate with
                                               the network interface. 

* `internal_dns_name_label` - (Optional) Relative DNS name for this NIC used for internal communications between VMs in the same VNet

* `dns_servers` - (Optional) List of DNS servers IP addresses to use for this NIC, overrides the VNet-level server list

* `ip_configuration` - (Optional) Collection of ipConfigurations associated with this NIC. Each `ip_configuration` block supports fields documented below.

The `ip_configuration` block supports:

* `name` - (Required) User-defined name of the IP.

* `subnet_id` - (Required) Reference to a subnet in which this NIC has been created.

* `private_ip_address` - (Optional) Static IP Address.

* `private_ip_address_allocation` - (Required) Defines how a private IP address is assigned. Options are Static or Dynamic.

* `public_ip_address_id` - (Optional) Reference to a Public IP Address to associate with this NIC

* `load_balancer_backend_address_pools_ids` - (Optional) List of Load Balancer Backend Address Pool IDs references to which this NIC belongs

* `load_balancer_inbound_nat_rules_ids` - (Optional) List of Load Balancer Inbound Nat Rules IDs involving this NIC

## Attributes Reference

The following attributes are exported:

* `id` - The virtual NetworkConfiguration ID.
* `mac_address` - 
* `virtual_machine_id` - 
* `applied_dns_servers` - 
* `internal_fqdn` - 
