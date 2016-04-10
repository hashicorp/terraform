---
layout: "google"
page_title: "Google: google_sql_database"
sidebar_current: "docs-google-sql-database"
description: |-
  Creates a new SQL database in Google Cloud SQL.
---

# google\_sql\_database

Creates a new Google SQL Database on a Google SQL Database Instance. For more information, see the [official documentation](https://cloud.google.com/sql/), or the [JSON API](https://cloud.google.com/sql/docs/admin-api/v1beta4/databases).

## Example Usage

Example creating a SQL Database.

```js
resource "google_sql_database_instance" "master" {
  name = "master-instance"

  settings {
    tier = "D0"
  }
}

resource "google_sql_database" "users" {
  name     = "image-store-bucket"
  instance = "${google_sql_database_instance.master.name}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the database.

* `instance` - (Required) The name of containing instance.

- - -

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.
