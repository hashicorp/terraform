---
layout: "mssql"
page_title: "Provider: MSSQL"
sidebar_current: "docs-mssql-index"
description: |-
  A provider for MS SQL Server.
---

# MSSQL Provider

The MSSQL provider gives the ability to deploy and configure resources in a MS SQL server.

Use the navigation to the left to read about the available resources.

## Usage

```
provider "mssql" {
  host = "ms_sql_server_endpoint"
  port = 1433
  username = "mssql_user"
  password = "mssql_password"
}

```

Configuring multiple servers can be done by specifying the alias option.

```
provider "mssql" {
  alias = "sql1"
  host = "ms_sql_server_endpoint1"
  username = "mssql_user1"
  password = "mssql_password1"
}

provider "mssql" {
  alias = "sql2"
  host = "ms_sql_server_endpoint2"
  username = "mssql_user2"
  password = "mssql_password2"
}

resource "mssql_database" "my_db1" {
  provider = "mssql.sql1"
  name = "my_db1"
}
resource "mssql_database" "my_db2" {
  provider = "mssql.sql2"
  name = "my_db2"
}


```

## Argument Reference

The following arguments are supported:

* `host` - (Required) The address for the MS SQL server connection.
* `port` - (Optional) The port for the MS SQL server connection. (Default 1433)
* `username` - (Required) Username for the server connection.
* `password` - (Required) Password for the server connection.
* `encrypt` - (Optional) Set to true if data sent between client and server should be encrypted.
  Available values are 'true', 'false', 'disable'. (Default 'false')
* `trust_server_certificate` - (Optional) Set to false if server certificate should be checked.
  If trust server certificate is true, driver accepts any certificate presented by the server and
  any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.
  This should be used only for testing. (Default true)
* `certificate` - (Optional) Full path to the file that contains the public key certificate of
  the CA that signed the SQL Server certificate.
