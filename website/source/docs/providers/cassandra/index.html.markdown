---
layout: "cassandra"
page_title: "Provider: Cassandra"
sidebar_current: "docs-cassandra-index"
description: |-
  The Cassandra provider configures keyspaces on a Cassandra server.
---

# Cassandra Provider

The Cassandra provider allows Terraform to create Keyspaces in
[Cassandra](http://cassandra.apache.org/).  Cassandra database is the right choice when you need scalability and
high availability without compromising performance

The provider configuration block accepts the following arguments:

* ``hostport`` - (Optional) The dns address and port of a Cassandra server. May alternatively be
  set via the ``CASSANDRA_HOSTPORT`` environment variable. Defaults to `localhost:9042`.

* ``username`` - (Optional) The name of the user to use when making requests.
  May alternatively be set via the ``CASSANDRA_USERNAME`` environment variable.

* ``password`` - (Optional) The password to use when making requests.
  May alternatively be set via the ``CASSANDRA_PASSWORD`` environment variable.

* ``proto_version`` - (Optional) The wire protocol version to use when making requests.
  May alternatively be set via the ``CASSANDRA_PROTO_VERSION`` environment variable.
  Defaults to ``3``


Use the navigation to the left to read about the available resources.

## Example Usage

```
provider "cassandra" {
    host_port = "localhost:9042"
    username = "terraform"
}

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
