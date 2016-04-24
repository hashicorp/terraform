---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_dns_aaaa_record"
sidebar_current: "docs-azurerm-resource-dns-aaaa-record"
description: |-
  Create a DNS AAAA Record.
---

# azurerm\_dns\_aaaa\_record

Enables you to manage DNS AAAA Records within Azure DNS.

## Example Usage

```
resource "azurerm_resource_group" "test" {
   name = "acceptanceTestResourceGroup1"
   location = "West US"
}
resource "azurerm_dns_zone" "test" {
   name = "mydomain.com"
   resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_aaaa_record" "test" {
   name = "test"
   zone_name = "${azurerm_dns_zone.test.name}"
   resource_group_name = "${azurerm_resource_group.test.name}"
   ttl = "300"
   records = ["2607:f8b0:4009:1803::1005"]
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DNS AAAA Record.

* `resource_group_name` - (Required) Specifies the resource group where the resource exists. Changing this forces a new resource to be created.

* `zone_name` - (Required) Specifies the DNS Zone where the resource exists. Changing this forces a new resource to be created.

* `TTL` - (Required) The Time To Live (TTL) of the DNS record.

* `records` - (Required) List of IPv6 Addresses.

* `tags` - (Optional) A mapping of tags to assign to the resource. 

## Attributes Reference

The following attributes are exported:

* `id` - The DNS AAAA Record ID.
