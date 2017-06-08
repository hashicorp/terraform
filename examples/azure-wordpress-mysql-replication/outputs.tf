output "resource_group" {
  value = "${var.resource_group}"
}

output "fqdn" {
  value = "${azurerm_public_ip.pip.fqdn}:3306"
}

output "ip_address" {
  value = "${azurerm_public_ip.pip.ip_address}"
}

output "ssh_command" {
  value = "ssh ${var.vm_admin_username}@${azurerm_public_ip.pip.ip_address} -p 64001"
}
