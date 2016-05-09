---
layout: "azurerm"
page_title: "Azure Resource Manager: azure_network_interface"
sidebar_current: "docs-azurerm-resource-network-interface"
description: |-
  Manages the Network Interface cards that link the Virtual Machines and Virtual Network.
---

# azurerm\_network\_interface

Network interface cards are virtual network cards that form the link between virtual machines and the virtual network

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
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
    name = "acceptanceTestNetworkInterface1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    ip_configuration {
    	name = "testconfiguration1"
    	subnet_id = "${azurerm_subnet.test.id}"
    	private_ip_address_allocation = "dynamic"
    }

    tags {
	    environment = "staging"
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

* `tags` - (Optional) A mapping of tags to assign to the resource. 

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
* `mac_address` - The media access control (MAC) address of the network interface.
* `private_ip_address` - The private ip address of the network interface.
* `virtual_machine_id` - Reference to a VM with which this NIC has been associated.
* `applied_dns_servers` - If the VM that uses this NIC is part of an Availability Set, then this list will have the union of all DNS servers from all NICs that are part of the Availability Set
* `internal_fqdn` - Fully qualified DNS name supporting internal communications between VMs in the same VNet
