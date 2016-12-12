---
layout: "postgresql"
page_title: "PostgreSQL: postgresql_schema"
sidebar_current: "docs-postgresql-resource-postgresql_schema"
description: |-
  Creates and manages a schema within a PostgreSQL database.
---

# postgresql\_schema

The ``postgresql_schema`` resource creates and manages a schema within a
PostgreSQL database.


## Usage

```
resource "postgresql_schema" "my_schema" {
  name = "my_schema"
  authorization = "my_role"
}
```

## Argument Reference

* `name` - (Required) The name of the schema. Must be unique in the PostgreSQL
  database instance where it is configured.

* `authorization` - (Optional) The owner of the schema.  Defaults to the
  username configured in the schema's provider.

## Import Example

`postgresql_schema` supports importing resources.  Supposing the following
Terraform:

```
provider "postgresql" {
  alias = "admindb"
}

resource "postgresql_schema" "schema_foo" {
  provider = "postgresql.admindb"

  name = "my_schema"
}
```

It is possible to import a `postgresql_schema` resource with the following
command:

```
$ terraform import postgresql_schema.schema_foo my_schema
```

Where `my_schema` is the name of the schema in the PostgreSQL database and
`postgresql_schema.schema_foo` is the name of the resource whose state will be
populated as a result of the command.
