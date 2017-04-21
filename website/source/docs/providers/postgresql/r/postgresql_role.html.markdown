---
layout: "postgresql"
page_title: "PostgreSQL: postgresql_role"
sidebar_current: "docs-postgresql-resource-postgresql_role"
description: |-
  Creates and manages a role on a PostgreSQL server.
---

# postgresql\_role

The ``postgresql_role`` resource creates and manages a role on a PostgreSQL
server.

When a ``postgresql_role`` resource is removed, the PostgreSQL ROLE will
automatically run a [`REASSIGN
OWNED`](https://www.postgresql.org/docs/current/static/sql-reassign-owned.html)
and [`DROP
OWNED`](https://www.postgresql.org/docs/current/static/sql-drop-owned.html) to
the `CURRENT_USER` (normally the connected user for the provider).  If the
specified PostgreSQL ROLE owns objects in multiple PostgreSQL databases in the
same PostgreSQL Cluster, one PostgreSQL provider per database must be created
and all but the final ``postgresql_role`` must specify a `skip_drop_role`.

~> **Note:** All arguments including role name and password will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Usage

```hcl
resource "postgresql_role" "my_role" {
  name     = "my_role"
  login    = true
  password = "mypass"
}

resource "postgresql_role" "my_replication_role" {
  name             = "replication_role"
  replication      = true
  login            = true
  connection_limit = 5
  password         = "md5c98cbfeb6a347a47eb8e96cfb4c4b890"
}
```

## Argument Reference

* `name` - (Required) The name of the role. Must be unique on the PostgreSQL
  server instance where it is configured.

* `superuser` - (Optional) Defines whether the role is a "superuser", and
  therefore can override all access restrictions within the database.  Default
  value is `false`.

* `create_database` - (Optional) Defines a role's ability to execute `CREATE
  DATABASE`.  Default value is `false`.

* `create_role` - (Optional) Defines a role's ability to execute `CREATE ROLE`.
  A role with this privilege can also alter and drop other roles.  Default value
  is `false`.

* `inherit` - (Optional) Defines whether a role "inherits" the privileges of
  roles it is a member of.  Default value is `true`.

* `login` - (Optional) Defines whether role is allowed to log in.  Roles without
  this attribute are useful for managing database privileges, but are not users
  in the usual sense of the word.  Default value is `false`.

* `replication` - (Optional) Defines whether a role is allowed to initiate
  streaming replication or put the system in and out of backup mode.  Default
  value is `false`

* `bypass_row_level_security` - (Optional) Defines whether a role bypasses every
  row-level security (RLS) policy.  Default value is `false`.

* `connection_limit` - (Optional) If this role can log in, this specifies how
  many concurrent connections the role can establish. `-1` (the default) means no
  limit.

* `encrypted_password` - (Optional) Defines whether the password is stored
  encrypted in the system catalogs.  Default value is `true`.  NOTE: this value
  is always set (to the conservative and safe value), but may interfere with the
  behavior of
  [PostgreSQL's `password_encryption` setting](https://www.postgresql.org/docs/current/static/runtime-config-connection.html#GUC-PASSWORD-ENCRYPTION).

* `password` - (Optional) Sets the role's password. (A password is only of use
  for roles having the `login` attribute set to true, but you can nonetheless
  define one for roles without it.) Roles without a password explicitly set are
  left alone.  If the password is set to the magic value `NULL`, the password
  will be always be cleared.

* `valid_until` - (Optional) Defines the date and time after which the role's
  password is no longer valid.  Established connections past this `valid_time`
  will have to be manually terminated.  This value corresponds to a PostgreSQL
  datetime. If omitted or the magic value `NULL` is used, `valid_until` will be
  set to `infinity`.  Default is `NULL`, therefore `infinity`.

* `skip_drop_role` - (Optional) When a PostgreSQL ROLE exists in multiple
  databases and the ROLE is dropped, the
  [cleanup of ownership of objects](https://www.postgresql.org/docs/current/static/role-removal.html)
  in each of the respective databases must occur before the ROLE can be dropped
  from the catalog.  Set this option to true when there are multiple databases
  in a PostgreSQL cluster using the same PostgreSQL ROLE for object ownership.
  This is the third and final step taken when removing a ROLE from a database.

* `skip_reassign_owned` - (Optional) When a PostgreSQL ROLE exists in multiple
  databases and the ROLE is dropped, a
  [`REASSIGN OWNED`](https://www.postgresql.org/docs/current/static/sql-reassign-owned.html) in
  must be executed on each of the respective databases before the `DROP ROLE`
  can be executed to dropped the ROLE from the catalog.  This is the first and
  second steps taken when removing a ROLE from a database (the second step being
  an implicit
  [`DROP OWNED`](https://www.postgresql.org/docs/current/static/sql-drop-owned.html)).

## Import Example

`postgresql_role` supports importing resources.  Supposing the following
Terraform:

```hcl
provider "postgresql" {
  alias = "admindb"
}

resource "postgresql_role" "replication_role" {
  provider = "postgresql.admindb"

  name = "replication_name"
}
```

It is possible to import a `postgresql_role` resource with the following
command:

```
$ terraform import postgresql_role.replication_role replication_name
```

Where `replication_name` is the name of the role to import and
`postgresql_role.replication_role` is the name of the resource whose state will
be populated as a result of the command.
