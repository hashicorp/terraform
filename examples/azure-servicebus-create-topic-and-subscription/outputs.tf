output "NamespaceConnectionString" {
  value = "${azurerm_servicebus_namespace.test.default_primary_connection_string}"
}

output "SharedAccessPolicyPrimaryKey" {
  value = "${azurerm_servicebus_namespace.test.default_primary_key}"
}
