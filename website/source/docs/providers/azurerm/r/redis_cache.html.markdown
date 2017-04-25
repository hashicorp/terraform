---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_redis_cache"
sidebar_current: "docs-azurerm-resource-redis-cache"
description: |-
  Creates a new Redis Cache Resource
---

# azurerm\_redis\_cache

Creates a new Redis Cache Resource

## Example Usage (Basic)

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acceptanceTestResourceGroup1"
  location = "West US"
}

resource "azurerm_redis_cache" "test" {
  name                = "test"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  capacity            = 0
  family              = "C"
  sku_name            = "Basic"
  enable_non_ssl_port = false

  redis_configuration {
    maxclients = "256"
  }
}
```

## Example Usage (Standard)

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acceptanceTestResourceGroup1"
  location = "West US"
}

resource "azurerm_redis_cache" "test" {
  name                = "test"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  capacity            = 2
  family              = "C"
  sku_name            = "Standard"
  enable_non_ssl_port = false

  redis_configuration {
    maxclients = "1000"
  }
}
```

## Example Usage (Premium with Clustering)

```hcl
resource "azurerm_resource_group" "test" {
  name     = "acceptanceTestResourceGroup1"
  location = "West US"
}

resource "azurerm_redis_cache" "test" {
  name                = "clustered-test"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  capacity            = 1
  family              = "P"
  sku_name            = "Premium"
  enable_non_ssl_port = false
  shard_count         = 3

  redis_configuration {
    maxclients         = "7500"
    maxmemory_reserved = "2"
    maxmemory_delta    = "2"
    maxmemory_policy   = "allkeys-lru"
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

* `capacity` - (Required) The size of the Redis cache to deploy. Valid values for a SKU `family` of C (Basic/Standard) are `0, 1, 2, 3, 4, 5, 6`, and for P (Premium) `family` are `1, 2, 3, 4`.

* `family` - (Required) The SKU family to use. Valid values are `C` and `P`, where C = Basic/Standard, P = Premium.

The pricing group for the Redis Family - either "C" or "P" at present.

* `sku_name` - (Required) The SKU of Redis to use - can be either Basic, Standard or Premium.

* `enable_non_ssl_port` - (Optional) Enable the non-SSL port (6789) - disabled by default.

* `shard_count` - (Optional) *Only available when using the Premium SKU* The number of Shards to create on the Redis Cluster.

* `redis_configuration` - (Required) Potential Redis configuration values - with some limitations by SKU - defaults/details are shown below.

```hcl
redis_configuration {
  maxclients         = "512"
  maxmemory_reserve  = "10"
  maxmemory_delta    = "2"
  maxmemory_policy   = "allkeys-lru"
}
```

## Default Redis Configuration Values
| Redis Value        | Basic        | Standard     | Premium      |
| ------------------ | ------------ | ------------ | ------------ |
| maxclients         | 256          | 1000         | 7500         |
| maxmemory_reserved | 2            | 50           | 200          |
| maxmemory_delta    | 2            | 50           | 200          |
| maxmemory_policy   | volatile-lru | volatile-lru | volatile-lru |

_*Important*: The maxmemory_reserved setting is only available for Standard and Premium caches. More details are available in the Relevant Links section below._

## Attributes Reference

The following attributes are exported:

* `id` - The Route ID.

* `hostname` - The Hostname of the Redis Instance

* `ssl_port` - The SSL Port of the Redis Instance

* `port` - The non-SSL Port of the Redis Instance

* `primary_access_key` - The Primary Access Key for the Redis Instance

* `secondary_access_key` - The Secondary Access Key for the Redis Instance

## Relevant Links
 - [Azure Redis Cache: SKU specific configuration limitations](https://azure.microsoft.com/en-us/documentation/articles/cache-configure/#advanced-settings)
 - [Redis: Available Configuration Settings](http://redis.io/topics/config)
