---
layout: "azure"
page_title: "Azure: azure_sql_database_server"
sidebar_current: "docs-azure-sql-database-server"
description: |-
    Allocates a new SQL Database Server on Azure.
---

# azure\_sql\_database\_server

Allocates a new SQL Database Server on Azure.

## Example Usage

```
resource "azure_sql_database_server" "sql-serv" {
    name = "<computed>"
    location = "West US"
    username = "SuperUser"
    password = "SuperSEKR3T"
    version = "2.0"
    url = "<computed>"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Computed) The name of the database server. It is determined upon
    creation as it is randomly-generated per server.

* `location` - (Required) The location where the database server should be created.
    For a list of all Azure locations, please consult [this link](http://azure.microsoft.com/en-us/regions/).

* `username` - (Required) The username for the administrator of the database server.

* `password` - (Required) The password for the administrator of the database server.

* `version` - (Optional) The version of the database server to be used. Can be any
    one of `2.0` or `12.0`.

* `url` - (Computed) The fully qualified domain name of the database server.
    Will be of the form `<name>.database.windows.net`.

## Attributes Reference

The following attributes are exported:

* `id` - The database server ID. Coincides with the randomly-generated `name`.
