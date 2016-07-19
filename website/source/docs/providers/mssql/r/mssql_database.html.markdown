---
layout: "mssql"
page_title: "MSSQL: mssql_database"
sidebar_current: "docs-mssql-resource-mssql_database"
description: |-
  Creates and manages a database on a MS SQL server.
---

# mssql\_database

The ``mssql_database`` resource creates and manages a database on a MS SQL server.


## Usage

```
resource "mssql_database" "my_db" {
   name = "my_db"
}

```

## Argument Reference

* `name` - (Required) The name of the database. Must be unique on the MS SQL server instance
  where it is configured.
