---
layout: "postgresql"
page_title: "PostgreSQL: postgresql_role"
sidebar_current: "docs-postgresql-resource-postgresql_role"
description: |-
  Creates and manages a database on a PostgreSQL server.
---

# postgresql\_role

The ``postgresql_role`` resource creates and manages a role on a PostgreSQL
server.


## Usage

```
resource "postgresql_role" "my_role" {
  name = "my_role"
  login = true
  password = "mypass"
  encrypted = true
}

```

## Argument Reference

* `name` - (Required) The name of the role. Must be unique on the PostgreSQL server instance
  where it is configured.

* `login` - (Optional) Configures whether a role is allowed to log in; that is, whether the role can be given as the initial session authorization name during client connection. Corresponds to the LOGIN/NOLOGIN
clauses in 'CREATE ROLE'. Default value is false.

* `password` - (Optional) Sets the role's password. (A password is only of use for roles having the LOGIN attribute, but you can nonetheless define one for roles without it.) If you do not plan to use password authentication you can omit this option. If no password is specified, the password will be set to null and password authentication will always fail for that user.

* `encrypted` - (Optional) Corresponds to ENCRYPTED, UNENCRYPTED in PostgreSQL. This controls whether the password is stored encrypted in the system catalogs. Default is false.