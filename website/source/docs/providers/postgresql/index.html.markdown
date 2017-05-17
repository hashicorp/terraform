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

```hcl
provider "postgresql" {
  host            = "postgres_server_ip"
  port            = 5432
  database        = "postgres"
  username        = "postgres_user"
  password        = "postgres_password"
  sslmode         = "require"
  connect_timeout = 15
}
```

Configuring multiple servers can be done by specifying the alias option.

```hcl
provider "postgresql" {
  alias    = "pg1"
  host     = "postgres_server_ip1"
  username = "postgres_user1"
  password = "postgres_password1"
}

provider "postgresql" {
  alias    = "pg2"
  host     = "postgres_server_ip2"
  username = "postgres_user2"
  password = "postgres_password2"
}

resource "postgresql_database" "my_db1" {
  provider = "postgresql.pg1"
  name     = "my_db1"
}

resource "postgresql_database" "my_db2" {
  provider = "postgresql.pg2"
  name     = "my_db2"
}
```

## Argument Reference

The following arguments are supported:

* `host` - (Required) The address for the postgresql server connection.
* `port` - (Optional) The port for the postgresql server connection. The default is `5432`.
* `database` - (Optional) Database to connect to. The default is `postgres`.
* `username` - (Required) Username for the server connection.
* `password` - (Optional) Password for the server connection.
* `sslmode` - (Optional) Set the priority for an SSL connection to the server.
  Valid values for `sslmode` are (note: `prefer` is not supported by Go's
  [`lib/pq`](https://godoc.org/github.com/lib/pq)):
    * disable - No SSL
    * require - Always SSL (the default, also skip verification)
    * verify-ca - Always SSL (verify that the certificate presented by the server was signed by a trusted CA)
    * verify-full - Always SSL (verify that the certification presented by the server was signed by a trusted CA and the server host name matches the one in the certificate)
  Additional information on the options and their implications can be seen
  [in the `libpq(3)` SSL guide](http://www.postgresql.org/docs/current/static/libpq-ssl.html#LIBPQ-SSL-PROTECTION).
* `connect_timeout` - (Optional) Maximum wait for connection, in seconds. The
  default is `180s`.  Zero or not specified means wait indefinitely.
