---
layout: "azure"
page_title: "Azure: azure_sql_database_service"
sidebar_current: "docs-azure-sql-database-service"
description: |-
    Creates a new SQL Database Service on an Azure Database Server.
---

# azure\_sql\_database\_service

Creates a new SQL database service on an Azure database server.

## Example Usage

```
resource "azure_sql_database_service" "sql-server" {
    name = "terraform-testing-db-renamed"
    database_server_name = "flibberflabber"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "5368709120"
    service_level_id = "f1173c43-91bd-4aaa-973c-54e79e15235b"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the database service.

* `database_server_name` - (Required) The name of the database server this service
    should run on. Changes here force the creation of a new resource.

* `edition` - (Optional) The edition of the database service. For more information
    on each variant, please view [this](https://msdn.microsoft.com/library/azure/dn741340.aspx) link.

* `collation` - (Optional) The collation to be used within the database service.
    Defaults to the standard Latin charset.

* `max_size_bytes` - (Optional) The maximum size in bytes the database service
    should be allowed to expand to. Range depends on the database `edition`
    selected above.

* `service_level_id` - (Optional) The ID corresponding to the service level per
    edition. Please refer to [this](https://msdn.microsoft.com/en-us/library/azure/dn505701.aspx) link for more details.

## Attributes Reference

The following attributes are exported:

* `id` - The database service ID. Coincides with the given `name`.
