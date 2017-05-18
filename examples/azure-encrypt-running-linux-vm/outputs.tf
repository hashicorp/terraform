output "hostname" {
  value = "${var.hostname}"
}

output "vm_fqdn" {
  value = "${azurerm_public_ip.pip.fqdn}"
}

output "ssh_command" {
  value = "${concat("ssh ", var.admin_username, "@", azurerm_public_ip.pip.fqdn)}"
}

output "BitLockerKey" {
  value     = "${azurerm_template_deployment.linux_vm.outputs["BitLockerKey"]}"
  sensitive = true
}
