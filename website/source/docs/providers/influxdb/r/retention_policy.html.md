---
layout: "influxdb"
page_title: "InfluxDB: influxdb_retention_policy"
sidebar_current: "docs-influxdb-resource-retention_policy"
description: |-
  The influxdb_retention_policy resource allows an InfluxDB retention policy to be managed.
---

# influxdb\_retention\_policy

The retention_policy resource allows a retention policy to be created on an InfluxDB server.

## Example Usage

```
resource "influxdb_database" "test" {
    name = "terraform-test"
}

resource "influxdb_retention_policy" "policy" {
    name = "policy"
    database = "${influxdb_database.test.name}"
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name for the retention_policy. This must be unique for the specified influxdb database.
* `database` - (Required) The database for the retention_policy. This must be an existing influxdb database.
* `duration` - (Optional) The duration for how long InfluxDB keeps the data.
* `is_default` - (Optional) Whether or not this retention policy is the default for the given database. 
* `replication` - (Optional) Specifies the number of indepedent copies of data when clustering. 
* `shard_duration` - (Optional) Specifies the time range covered by a shard group when clustering.

## Attributes Reference

This resource exports no further attributes.
