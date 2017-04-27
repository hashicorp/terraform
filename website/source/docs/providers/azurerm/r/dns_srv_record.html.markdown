---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_dns_srv_record"
sidebar_current: "docs-azurerm-resource-dns-srv-record"
description: |-
  Manage a DNS SRV Record.
---

# azurerm\_dns\_srv\_record

Enables you to manage DNS SRV Records within Azure DNS.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acceptanceTestResourceGroup1"
  location = "West US"
}

resource "azurerm_dns_zone" "test" {
  name                = "mydomain.com"
  resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_dns_srv_record" "test" {
  name                = "test"
  zone_name           = "${azurerm_dns_zone.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  ttl                 = "300"

  record {
    priority = 1
    weight   = 5
    port     = 8080
    target   = "target1.contoso.com"
  }

  tags {
    Environment = "Production"
  }
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DNS SRV Record.

* `resource_group_name` - (Required) Specifies the resource group where the resource exists. Changing this forces a new resource to be created.

* `zone_name` - (Required) Specifies the DNS Zone where the resource exists. Changing this forces a new resource to be created.

* `TTL` - (Required) The Time To Live (TTL) of the DNS record.

* `record` - (Required) A list of values that make up the SRV record. Each `record` block supports fields documented below.

* `tags` - (Optional) A mapping of tags to assign to the resource.

The `record` block supports:

* `priority` - (Required) Priority of the SRV record.

* `weight` - (Required) Weight of the SRV record.

* `port` - (Required) Port the service is listening on.

* `target` - (Required) FQDN of the service.


## Attributes Reference

The following attributes are exported:

* `id` - The DNS SRV Record ID.

## Import

SRV records can be imported using the `resource id`, e.g.

```
terraform import azurerm_dns_srv_record.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/dnsZones/zone1/SRV/myrecord1
```
