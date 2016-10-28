---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_redis"
sidebar_current: "docs-azurerm-resource-redis"
description: |-
  Creates a new Redis Cache Resource
---

# azurerm\_redis

Creates a new Redis Cache Resource

## Example Usage (Basic)

```
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_redis" "test" {
  name                = "test"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  redis_version       = "3.0"
  capacity            = 0
  family              = "C"
  sku_name            = "Basic"
  enable_non_ssl_port = false
}

```

## Example Usage (Standard)

```
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_redis" "test" {
  name                = "test"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  redis_version       = "3.0"
  capacity            = 1
  family              = "C"
  sku_name            = "Standard"
  enable_non_ssl_port = false
}

```

## Example Usage (Premium with Clustering)
```
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_redis" "test" {
  name                = "clustered-test"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  redis_version       = "3.0"
  capacity            = 1
  family              = "C"
  sku_name            = "Premium"
  enable_non_ssl_port = false
  shard_count         = 3
  redis_configuration {
    "maxclients"         = "256",
    "maxmemory-reserved" = "2",
    "maxmemory-delta"    = "2"
    "maxmemory-policy"   = "allkeys-lru"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Redis instance. Changing this forces a
    new resource to be created.

* `location` - (Required) The location of the resource group.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the Redis instance.

* `redis_version` - (Required) The version of Redis to use.

* `capacity` - (Required) The amount of Redis Capacity / Storage required in GB. If you're using the Basic (250mb) tier, this value should be `0`.

* `family` - (Required) The pricing group for the Redis Family - either "C" or "P" at present.

* `sku_name` - (Required) The SKU of Redis to use - can be either Basic, Standard or Premium.

* `enable_non_ssl_port` - (Optional) Enable the non-SSL port (6789) - disabled by default.

* `redis_configuration` - (Optional) Any Redis configuration variables you might want to set.

* `shard_count` - (Optional) *Only available when using the Premium SKU* The number of Shards to create on the Redis Cluster.

## Attributes Reference

The following attributes are exported:

* `id` - The Route ID.

* `hostname` - The Hostname of the Redis Instance

* `ssl_port` - The non-SSL Port of the Redis Instance

* `port` - The SSL Port of the Redis Instance

* `primary_access_key` - The Primary Access Key for the Redis Instance

* `secondary_access_key` - The Secondary Access Key for the Redis Instance
