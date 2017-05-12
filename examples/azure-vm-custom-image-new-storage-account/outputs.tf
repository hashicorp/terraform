output "hostname" {
  value = "${var.hostname}"
}

output "ip_address" {
  value = "${azurerm_public_ip.transferpip.ip_address}"
}

output "fqdn" {
  value = "${azurerm_public_ip.transferpip.ip_address}"
}

output "id" {
  value = "${azurerm_public_ip.transferpip.id}"
}
