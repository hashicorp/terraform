---
layout: "postgresql"
page_title: "Postgresql: postgresql_role"
sidebar_current: "docs-postgresql-resource-postgresql_role"
description: |-
  Creates and manages a database on a Postgresql server.
---

# postgresql\postgresql_role

The ``postgresql_role`` resource creates and manages a role on a Postgresql
server.


## Usage

```
resource "postgresql_role" "my_role" {
  name = "my_role"
  login = true
}

```

## Argument Reference

* `name` - (Required) The name of the role. Must be unique on the Postgresql server instance
  where it is configured.

* `login` - (Optional) Configures whether a role is allowed to log in; that is, whether the role can be given as the initial session authorization name during client connection. Coresponds to the LOGIN/NOLOGIN
clauses in 'CREATE ROLE'. Default value is false.