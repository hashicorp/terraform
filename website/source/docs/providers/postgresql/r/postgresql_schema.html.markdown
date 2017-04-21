---
layout: "postgresql"
page_title: "PostgreSQL: postgresql_schema"
sidebar_current: "docs-postgresql-resource-postgresql_schema"
description: |-
  Creates and manages a schema within a PostgreSQL database.
---

# postgresql\_schema

The ``postgresql_schema`` resource creates and manages [schema
objects](https://www.postgresql.org/docs/current/static/ddl-schemas.html) within
a PostgreSQL database.


## Usage

```hcl
resource "postgresql_role" "app_www" {
  name = "app_www"
}

resource "postgresql_role" "app_dba" {
  name = "app_dba"
}

resource "postgresql_role" "app_releng" {
  name = "app_releng"
}

resource "postgresql_schema" "my_schema" {
  name  = "my_schema"
  owner = "postgres"

  policy {
    usage = true
    role  = "${postgresql_role.app_www.name}"
  }

  # app_releng can create new objects in the schema.  This is the role that
  # migrations are executed as.
  policy {
    create = true
    usage  = true
    role   = "${postgresql_role.app_releng.name}"
  }

  policy {
    create_with_grant = true
    usage_with_grant  = true
    role              = "${postgresql_role.app_dba.name}"
  }
}
```

## Argument Reference

* `name` - (Required) The name of the schema. Must be unique in the PostgreSQL
  database instance where it is configured.
* `owner` - (Optional) The ROLE who owns the schema.
* `policy` - (Optional) Can be specified multiple times for each policy.  Each
    policy block supports fields documented below.

The `policy` block supports:

* `create` - (Optional) Should the specified ROLE have CREATE privileges to the specified SCHEMA.
* `create_with_grant` - (Optional) Should the specified ROLE have CREATE privileges to the specified SCHEMA and the ability to GRANT the CREATE privilege to other ROLEs.
* `role` - (Optional) The ROLE who is receiving the policy.  If this value is empty or not specified it implies the policy is referring to the [`PUBLIC` role](https://www.postgresql.org/docs/current/static/sql-grant.html).
* `usage` - (Optional) Should the specified ROLE have USAGE privileges to the specified SCHEMA.
* `usage_with_grant` - (Optional) Should the specified ROLE have USAGE privileges to the specified SCHEMA and the ability to GRANT the USAGE privilege to other ROLEs.

~> **NOTE on `policy`:** The permissions of a role specified in multiple policy blocks is cumulative.  For example, if the same role is specified in two different `policy` each with different permissions (e.g. `create` and `usage_with_grant`, respectively), then the specified role with have both `create` and `usage_with_grant` privileges.

## Import Example

`postgresql_schema` supports importing resources.  Supposing the following
Terraform:

```hcl
resource "postgresql_schema" "public" {
  name = "public"
}

resource "postgresql_schema" "schema_foo" {
  name  = "my_schema"
  owner = "postgres"

  policy {
    usage = true
  }
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
