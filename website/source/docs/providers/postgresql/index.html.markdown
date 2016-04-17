---
layout: "postgresql"
page_title: "Provider: PostgreSQL"
sidebar_current: "docs-postgresql-index"
description: |-
  A provider for PostgreSQL Server.
---

# PostgreSQL Provider

The PostgreSQL provider gives the ability to deploy and configure resources in a PostgreSQL server.

Use the navigation to the left to read about the available resources.

## Usage

```
provider "postgresql" {
  host = "postgres_server_ip"
  port = 5432
  username = "postgres_user"
  password = "postgres_password"
  ssl_mode = "require"
}

```

Configuring multiple servers can be done by specifying the alias option.

```
provider "postgresql" {
  alias = "pg1"
  host = "postgres_server_ip1"
  username = "postgres_user1"
  password = "postgres_password1"
}

provider "postgresql" {
  alias = "pg2"
  host = "postgres_server_ip2"
  username = "postgres_user2"
  password = "postgres_password2"
}

resource "postgresql_database" "my_db1" {
  provider = "postgresql.pg1"
  name = "my_db1"
}
resource "postgresql_database" "my_db2" {
  provider = "postgresql.pg2"
  name = "my_db2"
}


```

## Argument Reference

The following arguments are supported:

* `host` - (Required) The address for the postgresql server connection.
* `port` - (Optional) The port for the postgresql server connection. The default is `5432`.
* `username` - (Required) Username for the server connection.
* `password` - (Optional) Password for the server connection.
* `ssl_mode` - (Optional) Set the priority for an SSL connection to the server.
  The default is `prefer`; the full set of options and their implications
  can be seen [in the libpq SSL guide](http://www.postgresql.org/docs/9.4/static/libpq-ssl.html#LIBPQ-SSL-PROTECTION).
