---
layout: "influxdb"
page_title: "Provider: InfluxDB"
sidebar_current: "docs-influxdb-index"
description: |-
  The InfluxDB provider configures databases, etc on an InfluxDB server.
---

# InfluxDB Provider

The InfluxDB provider allows Terraform to create Databases in
[InfluxDB](https://influxdb.com/). InfluxDB is a database server optimized
for time-series data.

The provider configuration block accepts the following arguments:

* ``url`` - (Optional) The root URL of a InfluxDB server. May alternatively be
  set via the ``INFLUXDB_URL`` environment variable. Defaults to
  `http://localhost:8086/`.

* ``username`` - (Optional) The name of the user to use when making requests.
  May alternatively be set via the ``INFLUXDB_USERNAME`` environment variable.

* ``password`` - (Optional) The password to use when making requests.
  May alternatively be set via the ``INFLUXDB_PASSWORD`` environment variable.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
provider "influxdb" {
  url      = "http://influxdb.example.com/"
  username = "terraform"
}

resource "influxdb_database" "metrics" {
  name = "awesome_app"
}

resource "influxdb_continuous_query" "minnie" {
  name     = "minnie"
  database = "${influxdb_database.metrics.name}"
  query    = "SELECT min(mouse) INTO min_mouse FROM zoo GROUP BY time(30m)"
}

resource "influxdb_user" "paul" {
  name     = "paul"
  password = "super-secret"
}
```
