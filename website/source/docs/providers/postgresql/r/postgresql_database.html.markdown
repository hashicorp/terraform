---
layout: "postgresql"
page_title: "PostgreSQL: postgresql_database"
sidebar_current: "docs-postgresql-resource-postgresql_database"
description: |-
  Creates and manages a database on a PostgreSQL server.
---

# postgresql\_database

The ``postgresql_database`` resource creates and manages a database on a PostgreSQL
server.


## Usage

```
resource "postgresql_database" "my_db" {
   name = "my_db"
   owner = "my_role"
}

```

## Argument Reference

* `name` - (Required) The name of the database. Must be unique on the PostgreSQL server instance
  where it is configured.

* `owner` - (Optional) The owner role of the database. If not specified the default is the user executing the command. To create a database owned by another role, you must be a direct or indirect member of that role, or be a superuser.
