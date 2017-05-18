output "hostname" {
  value = "${var.hostname}"
}

output "BitLockerKey" {
  value     = "${azurerm_template_deployment.linux_vm.outputs["BitLockerKey"]}"
  sensitive = true
}
