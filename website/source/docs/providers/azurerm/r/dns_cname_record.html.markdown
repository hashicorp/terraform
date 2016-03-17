---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_dns_cname_record"
sidebar_current: "docs-azurerm-resource-dns-cname-record"
description: |-
  Create a DNS CNAME Record.
---

# azurerm\_dns\_cname\_record

Enables you to manage DNS CNAME Records within Azure DNS.

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

resource "azurerm_dns_cname_record" "test" {
   name = "test"
   zone_name = "${azurerm_dns_zone.test.name}"
   resource_group_name = "${azurerm_resource_group.test.name}"
   ttl = "300"
   records = ["contoso.com"]
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DNS CNAME Record.

* `resource_group_name` - (Required) Specifies the resource group where the resource exists. Changing this forces a new resource to be created.

* `zone_name` - (Required) Specifies the DNS Zone where the resource exists. Changing this forces a new resource to be created.

* `TTL` - (Required) The Time To Live (TTL) of the DNS record.

* `records` - (Required) The target of the CNAME. Must be a single value.

* `tags` - (Optional) A mapping of tags to assign to the resource. 

## Attributes Reference

The following attributes are exported:

* `id` - The DNS CName Record ID.
