---
layout: "influxdb"
page_title: "InfluxDB: influxdb_database"
sidebar_current: "docs-influxdb-resource-database"
description: |-
  The influxdb_database resource allows an InfluxDB database to be created.
---

# influxdb\_database

The database resource allows a database to be created on an InfluxDB server.

## Example Usage

```hcl
resource "influxdb_database" "metrics" {
    name = "awesome_app"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name for the database. This must be unique on the
  InfluxDB server.

## Attributes Reference

This resource exports no further attributes.
