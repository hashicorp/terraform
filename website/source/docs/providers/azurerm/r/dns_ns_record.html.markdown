---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_dns_ns_record"
sidebar_current: "docs-azurerm-resource-dns-ns-record"
description: |-
  Create a DNS NS Record.
---

# azurerm\_dns\_ns\_record

Enables you to manage DNS NS Records within Azure DNS.

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

resource "azurerm_dns_ns_record" "test" {
   name = "test"
   zone_name = "${azurerm_dns_zone.test.name}"
   resource_group_name = "${azurerm_resource_group.test.name}"
   ttl = "300"
   record {
     nsdname = "ns1.contoso.com"
   }
   
   record {
     nsdname = "ns2.contoso.com"
   }
   
   tags {
     Environment = "Production"
   }
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DNS NS Record.

* `resource_group_name` - (Required) Specifies the resource group where the resource exists. Changing this forces a new resource to be created.

* `zone_name` - (Required) Specifies the DNS Zone where the resource exists. Changing this forces a new resource to be created.

* `TTL` - (Required) The Time To Live (TTL) of the DNS record.

* `record` - (Required) A list of values that make up the NS record. Each `record` block supports fields documented below.

* `tags` - (Optional) A mapping of tags to assign to the resource. 

The `record` block supports:

* `nsdname` - (Required) The value of the record.

## Attributes Reference

The following attributes are exported:

* `id` - The DNS NS Record ID.
