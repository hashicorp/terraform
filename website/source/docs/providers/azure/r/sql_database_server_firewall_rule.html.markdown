---
layout: "azure"
page_title: "Azure: azure_sql_database_server_firewall_rule"
sidebar_current: "docs-azure-sql-database-server-firewall-rule"
description: |-
    Defines a new Firewall Rule to be applied across the given Database Servers.
---

# azure\_sql\_database\_server

Defines a new Firewall Rule to be applied across the given Database Servers.

## Example Usage

```hcl
resource "azure_sql_database_server" "sql-serv1" {
  # ...
}

resource "azure_sql_database_server" "sql-serv2" {
  # ...
}

resource "azure_sql_database_server_firewall_rule" "constraint" {
  name     = "terraform-testing-rule"
  start_ip = "154.0.0.0"
  end_ip   = "154.0.0.255"

  database_server_names = [
    "${azure_sql_database_server.sql-serv1.name}",
    "${azure_sql_database_server.sql-serv2.name}",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the rule. Changing forces the creation of a
    new resource.

* `start_ip` - (Required) The IPv4 which will represent the lower bound of the
    rule's application IP's. Traffic to/from IP's greater than or equal to this
    one up to the `end_ip` will be permitted.

* `end_ip` - (Required) The IPv4 which will represent the upper bound of the
    rule's application IP's. Traffic to/from IP's lesser that or equal to this
    one all the way down to the `start_ip` will be permitted.

* `database_server_names` - (Required) The set of names of the Azure SQL
    Database servers the rule should be enforced on.

## Attributes Reference

The following attributes are exported:

* `id` - The database server ID. Coincides with the given `name`.
