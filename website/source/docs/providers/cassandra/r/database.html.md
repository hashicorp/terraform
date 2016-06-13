---
layout: "cassandra"
page_title: "Cassandra: cassandra_keyspace"
sidebar_current: "docs-cassandra-resource-keyspace"
description: |-
  The cassandra_keyspace resource creates cassandra keyspaces.
---

# cassandra\_keyspace

The keyspace resource creates a keyspace on a Cassandra server (or cluster).

## Example Usage

```
resource "cassandra_keyspace" "keyspace1" {
    name = "mykeyspacename"
    durable_writes = true
    replication_class = "SimpleStrategy"
    replication_factor = 2
}

resource "cassandra_keyspace" "keyspace2" {
    name = "otherkeyspacename"
    durable_writes = true
    replication_class = "NetworkTopologyStrategy"
    datacenters {
        dc1 = 2
        dc2 = 3
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name for the keyspace. This must be unique on the
  Cassandra server.
* `durable_writes` - (Required) An important Cassandra persistence configuration.
* `replication_class` - (Required) An important Cassandra availability configuration.
  One of [`SimpleStrategy, NetworkTopologyStrategy`].
* `replication_factor` -  (Required if using `SimpleStrategy`) An integer value for the number of replicas to maintain
* `datacenters` - (Required if using `NetworkTopologyStrategy`) A map of datacenter names to
  replication factor integer values
## Attributes Reference

This resource exports no further attributes.
