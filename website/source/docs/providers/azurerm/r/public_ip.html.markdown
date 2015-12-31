---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_public_ip"
sidebar_current: "docs-azurerm-resource-public-ip"
description: |-
    Creates a new Azure Public IP resource.
---

# azurerm\_public\_ip

Creates a new Azure Public IP resource.

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

* `name` - (Required) The name of the public IP. Changes force
    redeployment.

* `resource_group_name` - (Required) The name of the resource group in which
    the public IP should be created. Changes force redeployment.

* `location` - (Required) The location where the public IP should be created. Must
    be the same as the location of the resource group the public IP will belong to.
    For a list of all Azure locations, please consult [this link](http://azure.microsoft.com/en-us/regions/). Changes force redeployment.

* `dynamic_ip` - (Optional) Boolean flag to indicate whether a not an
    IP address should be given to the public IP through dhcp. Conflicts
    with `ip_address`. Using dynamic IP is the default behavior.

* `ip_address` - (Optional) The IP address to be used for
    the public IP. Conflicts with `dynamic_private_ip`.

* `dns_name` - (Required) Unique public DNS prefix for the deployment. The fqdn
  will look something like '<dnsname>.westus.cloudapp.azure.com'. Up to 62
  chars, digits or dashes, lowercase, should start with a letter: must conform
  to '^[a-z][a-z0-9-]{1,61}[a-z0-9]$'.

* `ip_config_id` - (Required) The ID of the IP Configuration to be used for
    the public IP.

* `fqdn` - (Computed) The fully-qualified domain name the public IP
  should use internally to refer to itself.

* `reverse_fqdn` - (Computed) The reverse fully qualified domain name
    for the public IP.

* `timeout` - (Optional) The positive number of minutes of laying idle
    that will trigger a timeout.

## Attributes Reference

The following attributes are exported:

* `id` - The public IP ID.
