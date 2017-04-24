---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_dns_txt_record"
sidebar_current: "docs-azurerm-resource-dns-txt-record"
description: |-
  Create a DNS TXT Record.
---

# azurerm\_dns\_txt\_record

Enables you to manage DNS TXT Records within Azure DNS.

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

resource "azurerm_dns_txt_record" "test" {
  name                = "test"
  zone_name           = "${azurerm_dns_zone.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  ttl                 = "300"

  record {
    value = "google-site-authenticator"
  }

  record {
    value = "more site information here"
  }

  tags {
    Environment = "Production"
  }
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DNS TXT Record.

* `resource_group_name` - (Required) Specifies the resource group where the resource exists. Changing this forces a new resource to be created.

* `zone_name` - (Required) Specifies the DNS Zone where the resource exists. Changing this forces a new resource to be created.

* `TTL` - (Required) The Time To Live (TTL) of the DNS record.

* `record` - (Required) A list of values that make up the txt record. Each `record` block supports fields documented below.

* `tags` - (Optional) A mapping of tags to assign to the resource.

The `record` block supports:

* `value` - (Required) The value of the record.

## Attributes Reference

The following attributes are exported:

* `id` - The DNS TXT Record ID.

## Import

TXT records can be imported using the `resource id`, e.g.

```
terraform import azurerm_dns_txt_record.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/dnsZones/zone1/TXT/myrecord1
```
