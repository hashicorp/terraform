---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_public_ip"
sidebar_current: "docs-azurerm-datasource-public-ip"
description: |-
  Get information about the specified public IP address.
---

# azurerm\_public\_ip

Use this data source to access the properties of an existing Azure Public IP Address.

## Example Usage

```hcl
data "azurerm_public_ip" "datasourceip" {
    name = "testPublicIp"
    resource_group_name = "acctestRG"
}

resource "azurerm_virtual_network" "helloterraformnetwork" {
    name = "acctvn"
    address_space = ["10.0.0.0/16"]
    location = "West US 2"
    resource_group_name = "acctestRG"
}

resource "azurerm_subnet" "helloterraformsubnet" {
    name = "acctsub"
    resource_group_name = "acctestRG"
    virtual_network_name = "${azurerm_virtual_network.helloterraformnetwork.name}"
    address_prefix = "10.0.2.0/24"
}

resource "azurerm_network_interface" "helloterraformnic" {
    name = "tfni"
    location = "West US 2"
    resource_group_name = "acctestRG"

    ip_configuration {
        name = "testconfiguration1"
        subnet_id = "${azurerm_subnet.helloterraformsubnet.id}"
        private_ip_address_allocation = "static"
        private_ip_address = "10.0.2.5"
        public_ip_address_id = "${data.azurerm_public_ip.datasourceip.id}"
    }
}
```

## Argument Reference

* `name` - (Required) Specifies the name of the public IP address.
* `resource_group_name` - (Required) Specifies the name of the resource group.


## Attributes Reference

* `domain_name_label` - The label for the Domain Name.
* `idle_timeout_in_minutes` - Specifies the timeout for the TCP idle connection.
* `fqdn` - Fully qualified domain name of the A DNS record associated with the public IP. This is the concatenation of the domainNameLabel and the regionalized DNS zone.
* `ip_address` - The IP address value that was allocated.
* `tags` - A mapping of tags to assigned to the resource.