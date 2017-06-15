output "resource_group" {
  value = "${var.resource_group}"
}

output "master_ip_address" {
  value = "${azurerm_public_ip.master.ip_address}"
}

output "master_ssh_command" {
  value = "ssh ${var.vm_admin_username}@${azurerm_public_ip.master.ip_address}"
}

output "master_web_ui_public_ip" {
  value = "${azurerm_public_ip.master.ip_address}:8080"
}
