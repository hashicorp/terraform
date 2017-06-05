---
layout: "influxdb"
page_title: "InfluxDB: influxdb_user"
sidebar_current: "docs-influxdb-resource-user"
description: |-
  The influxdb_user resource allows an InfluxDB users to be managed.
---

# influxdb\_user

The user resource allows a user to be created on an InfluxDB server.

## Example Usage

```hcl
resource "influxdb_database" "green" {
    name = "terraform-green"
}

resource "influxdb_user" "paul" {
    name = "paul"
    password = "super-secret"

    grant {
      database = "${influxdb_database.green.name}"
      privilege = "write"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name for the user. 
* `password` - (Required) The password for the user. 
* `admin` - (Optional) Mark the user as admin.
* `grant` - (Optional) A list of grants for non-admin users

Each `grant` supports the following:

* `database` - (Required) The name of the database the privilege is associated with
* `privilege` - (Required) The privilege to grant (READ|WRITE|ALL)

## Attributes Reference

* `admin` - (Bool) indication if the user is an admin or not.
