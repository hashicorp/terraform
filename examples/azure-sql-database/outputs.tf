output "database_name" {
  value = "${azurerm_sql_database.db.name}"
}

output "sql_server_fqdn" {
  value = "${azurerm_sql_server.server.fully_qualified_domain_name}"
}
