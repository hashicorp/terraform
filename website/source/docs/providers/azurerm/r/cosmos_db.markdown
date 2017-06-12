---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_cosmos_db"
sidebar_current: "docs-azurerm-resource-cosmos_db"
description: |-
  Creates a new CosmosDB (DocumentDB) Account.
---

# azurerm\_cosmos_db

Creates a new CosmosDB (DocumentDB) Account.

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "resourceGroup1"
    location = "West Europe"
}

resource "azurerm_cosmos_db" "test" {
  name                = "documentDBAccount1"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  offer_type          = "Standard"
  consistency_policy {
    consistency_level       = "BoundedStaleness"
    max_interval_in_seconds = 10
    max_staleness           = 200
  }

  failover_policy {
    location = "West Europe"
    priority = 0
  }

  failover_policy {
    location = "East US"
    priority = 1
  }

  tags {
    hello = "world"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the DocumentDB account. Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which the DocumentDB Account is created. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `offer_type` - (Required) Specifies the Offer Type to use for this DocumentDB Account - currently this can only be set to `Standard`.

* `consistency_policy` - (Required) Specifies a `consistency_policy` resource, used to define the consistency policy for this DocumentDB account.

* `failover_policy` - (Required) Specifies a `failover_policy` resource, used to define where data should be replicated.

* `ip_range_filter` - (Optional) DocumentDB Firewall Support: This value specifies the set of IP addresses or IP address ranges in CIDR form to be included as the allowed list of client IPs for a given database account. IP addresses/ranges must be comma separated and must not contain any spaces.

* `tags` - (Optional) A mapping of tags to assign to the resource.

`consistency_policy` supports the following:

* `consistency_level` - (Required) The Consistency Level to use for this DocumentDB Account - can be either `BoundedStaleness`, `Eventual`, `Session` or `Strong`.
* `max_interval_in_seconds` - (Optional) When used with the Bounded Staleness consistency level, this value represents the time amount of staleness (in seconds) tolerated. Accepted range for this value is 1 - 100. Required when `consistency_level` is set to `BoundedStaleness`.
* `max_staleness` - (Optional) When used with the Bounded Staleness consistency level, this value represents the number of stale requests tolerated. Accepted range for this value is 1 – 2,147,483,647. Required when `consistency_level` is set to `BoundedStaleness`.

`failover_policy` supports the following:

* `location` - (Required) The name of the Azure region to host replicated data.
* `priority` - (Required) The failover priority of the region. A failover priority of 0 indicates a write region. The maximum value for a failover priority = (total number of regions - 1). Failover priority values must be unique for each of the regions in which the database account exists.

## Attributes Reference

The following attributes are exported:

* `id` - The DocumentDB Account ID.

* `primary_master_key` - The Primary master key for the DocumentDB account.

* `secondary_master_key` - The Secondary master key for the DocumentDB account.

* `primary_readonly_master_key` - The Primary read-only master Key for the DocumentDB account.

* `secondary_readonly_master_key` - The Secondary read-only master key for the DocumentDB account.


## Import

CosmosDB accounts can be imported using the `resource id`, e.g.

```
terraform import azurerm_cosmos_db.account1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.DocumentDB/databaseAccounts/account1
```
