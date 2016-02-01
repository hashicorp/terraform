---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_public_ip"
sidebar_current: "docs-azurerm-resource-network-public-ip"
description: |-
  Create a Public IP Address.
---

# azurerm\_public\_ip

Create a Public IP Address.

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "resourceGroup1"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "acceptanceTestPublicIp1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
    
    tags {
        environment = "Production"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the availability set. Changing this forces a
    new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the availability set.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `public_ip_address_allocation` - (Required) Defines whether the IP address is stable or dynamic. Options are Static or Dynamic.

* `idle_timeout_in_minutes` - (Optional) Specifies the timeout for the TCP idle connection. The value can be set between 4 and 30 minutes.

* `domain_name_label` - (Optional) Label for the Domain Name. Will be used to make up the FQDN.  If a domain name label is specified, an A DNS record is created for the public IP in the Microsoft Azure DNS system.

* `reverse_fqdn` - (Optional) A fully qualified domain name that resolves to this public IP address. If the reverseFqdn is specified, then a PTR DNS record is created pointing from the IP address in the in-addr.arpa domain to the reverse FQDN.

* `tags` - (Optional) A mapping of tags to assign to the resource. 

## Attributes Reference

The following attributes are exported:

* `id` - The Public IP ID.
* `ip_address` - The IP address value that was allocated.
* `fqdn` - Fully qualified domain name of the A DNS record associated with the public IP. This is the concatenation of the domainNameLabel and the regionalized DNS zone