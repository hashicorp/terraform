---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_sql_firewall_rule"
sidebar_current: "docs-azurerm-resource-sql-firewall_rule"
description: |-
  Create a SQL Firewall Rule.
---

# azurerm\_sql\_firewall\_rule

Allows you to manage an Azure SQL Firewall Rule

## Example Usage

```
resource "azurerm_resource_group" "test" {
   name = "acceptanceTestResourceGroup1"
   location = "West US"
}
resource "azurerm_sql_database" "test" {
    name = "MySQLDatabase"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    

    tags {
    	environment = "production"
    }
}

resource "azurerm_sql_firewall_rule" "test" {
    name = "FirewallRule1"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    start_ip_address = "10.0.17.62"
    end_ip_address = "10.0.17.62"
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the SQL Server.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the sql server.

* `server_name` - (Required) The name of the SQL Server on which to create the Firewall Rule.

* `start_ip_address` - (Required) The starting IP address to allow through the firewall for this rule.

* `end_ip_address` - (Required) The ending IP address to allow through the firewall for this rule.

## Attributes Reference

The following attributes are exported:

* `id` - The SQL Firewall Rule ID.