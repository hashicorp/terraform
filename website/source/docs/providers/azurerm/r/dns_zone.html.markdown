---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_dns_zone"
sidebar_current: "docs-azurerm-resource-dns-zone"
description: |-
  Create a DNS Zone.
---

# azurerm\_dns\_zone

Enables you to manage DNS zones within Azure DNS. These zones are hosted on Azure's name servers to which you can delegate the zone from the parent domain.

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
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the DNS Zone. Must be a valid domain name.

* `resource_group_name` - (Required) Specifies the resource group where the resource exists. Changing this forces a new resource to be created.

* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The DNS Zone ID.
* `max_number_of_record_sets` - (Optional) Maximum number of Records in the zone. Defaults to `1000`.
* `number_of_record_sets` - (Optional) The number of records already in the zone.
* `name_servers` - (Optional) A list of values that make up the NS record for the zone.


## Import

DNS Zones can be imported using the `resource id`, e.g.

```
terraform import azurerm_dns_zone.zone1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Network/dnsZones/zone1
```
