---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_network_interface"
sidebar_current: "docs-azurerm-resource-network-interface"
description: |-
    Creates a new VM network interface on Azure.
---

# azurerm\_network\_interface

Creates a new VM network interface on Azure.

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_virtual_network" "test" {
    resource_group_name = "${azurerm_resource_group.test.name}"
    name = "acceptanceTestVirtualNetwork1"
    address_space = ["10.0.0.0/16"]
    location = "West US"

    subnet {
        name = "subnet1"
        address_prefix = "10.0.1.0/24"
    }
}

# TODO: resource "azurerm_instance" "test" ...

resource "azurerm_public_ip" "test" {
    name = "testAccPublicIPAddress1"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "${azurerm_resource_group.test.name}"
    dns_name = "testAccDnsName1"
    ip_config_id = "${azurerm_network_interface.test.ip_config.0.id}"
}

resource "azurerm_network_interface" "test" {
    resource_group_name = "${azurerm_resource_group.test.name}"
    name = "acceptanceTestPublicIPAddress1"
    location = "West US"
    vm_id = "${azurerm_instance.test.id}"
    # TODO: network_security_group_id = ...

    ip_config = {
        name = "acceptanceTestIpConfiguration1"
        dynamic_private_ip = true
        # TODO: subnet_id = "${azurerm_virtual_network.test.subnet.HASH.id}"
        public_ip_id = "${azurerm_public_ip.test.id}"
    }

    dns_servers = ["8.8.8.8", "8.8.4.4"]
    applied_dns_servers: ["8.8.8.8"]
    internal_name = "iface1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the network interface. Changes force
    redeployment.

* `resource_group_name` - (Required) The name of the resource group in which
    the network interface should be created. Changes force redeployment.

* `location` - (Required) The location where the network interface should be created. Must
    be the same as the location of the resource group the interface will belong to.
    For a list of all Azure locations, please consult [this link](http://azure.microsoft.com/en-us/regions/). Changes force redeployment.

* `vm_id` - (Required) The ID of the VM the interface will be attached to.
    Changes force redeployment.

* `mac_address` - (Computed) The MAC address of the network interface.

* `network_security_group_id` - (Optional) The ID of the network security group
    by which to govern trafic through the interface.

* `dns_servers` - (Optional) A list of addresses of DNS servers to be used.

* `applied_dns_servers` - (Optional) A list consisting of the addresses of DNS
    servers which get used for resolving addresses.

* `internal_name` - (Optional) The domain name the interface should use internally
    to refer to itself.

* `internal_fqdn` - (Optional) The fully-qualified domain name the interface
  should use internally to refer to itself.

* `ip_config` - (Required) A list of fields denoting an IP configuration for the
  interface. Can be declared multiple times for multiple configurations.

An `ip_config` definition contains the following:

* `id` - (Computed) The unique ID of the IP configuration.

* `name` - (Required) Name of the network interface IP configuration.

* `dynamic_private_ip` - (Optional) Boolean flag to indicate whether a not a
    private IP address should be given to the interface dynamically. Conflicts
    with `private_ip_address`. Is the default behavior.

* `private_ip_address` - (Optional) Address to be used as the private IP for
    the network interface. Conflicts with `dynamic_private_ip`.

* `subnet_id` - (Required) ID of the subnet which the network interface should
    be connected to.

* `public_ip_id` - (Optional) ID of the public IP which points to the network
    interface.

* `load_balancer_bakend_bool_ids` - (Optional) List of the string IDs of load
        balancer backend pools which rely on this public IP configuration.

* `load_balancer_inbound_nat_rule_ids` - (Optional) List of the string IDs of load
        balancer inbound NAT rules, which are enforced through this IP config.



## Attributes Reference

The following attributes are exported:

* `id` - The network interface ID.
