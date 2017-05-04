output "hostname" {
  value = "${var.vmss_name}"
}

output "vm_fqdn" {
  value = "${azurerm_public_ip.pip.fqdn}"
}

output "ssh_command" {
  value = "ssh ${var.admin_username}@${azurerm_public_ip.pip.fqdn}"
}