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
  name  = "my_schema"
  owner = "postgres"
}
```

## Argument Reference

* `name` - (Required) The name of the schema. Must be unique in the PostgreSQL
  database instance where it is configured.

* `owner` - (Optional) The ROLE who owns the schema.

## Import Example

`postgresql_schema` supports importing resources.  Supposing the following
Terraform:

```
resource "postgresql_schema" "public" {
  name  = "public"
}

resource "postgresql_schema" "schema_foo" {
  name  = "my_schema"
  owner = "postgres"
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
