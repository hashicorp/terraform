---
layout: "postgresql"
page_title: "PostgreSQL: postgresql_schema_policy"
sidebar_current: "docs-postgresql-resource-postgresql_schema_policy"
description: |-
  Manages the permissions of PostgreSQL schemas.
---

# postgresql\_schema\_policy

The ``postgresql_schema_policy`` resource applies the necessary SQL DCL
(`GRANT`s and `REVOKE`s) necessary to ensure access compliance to a particular
SCHEMA within a PostgreSQL database.


## Usage

```
resource "postgresql_role" "my_app" {
  name = "my_app"
}

resource "postgresql_schema" "my_schema" {
  name  = "my_schema"
}

resource "postgresql_schema_policy" "my_schema" {
  create = true
  usage = true
  schema = "${postgresql_schema.my_schema.name}"
  role = "${postgresql_role.my_app.name}"
}
```

## Argument Reference

* `create` - (Optional) Should the specified ROLE have CREATE privileges to the specified SCHEMA.

* `create_with_grant` - (Optional) Should the specified ROLE have CREATE privileges to the specified SCHEMA and the ability to GRANT the CREATE privilege to other ROLEs.

* `usage` - (Optional) Should the specified ROLE have USAGE privileges to the specified SCHEMA.

* `usage_with_grant` - (Optional) Should the specified ROLE have USAGE privileges to the specified SCHEMA and the ability to GRANT the USAGE privilege to other ROLEs.

* `role` - (Required) The ROLE who is receiving the policy.

* `schema` - (Required) The SCHEMA that is the target of the policy.

## Import Example

`postgresql_schema_policy` supports importing resources.  Supposing the
following Terraform:

```
resource "postgresql_schema" "public" {
  name  = "public"
}

resource "postgresql_schema_policy" "public" {
  create = true
  usage = true
  schema = "${postgresql_schema.public.name}"
  role = "${postgresql_role.my_app.name}"
}
```

It is possible to import a `postgresql_schema_policy` resource with the
following command:

```
$ terraform import postgresql_schema_policy.public public
```

Where `public` is the name of the schema in the PostgreSQL database and
`postgresql_schema_policy.public` is the name of the resource whose state will
be populated as a result of the command.
