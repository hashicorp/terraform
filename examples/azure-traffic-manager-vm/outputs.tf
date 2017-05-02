output "dns_name" {
  value = "${var.dns_name}"
}

# output "vm_fqdn" {
#   value = ["${element(azurerm_public_ip.pip.*.fqdn, count.index)}"]
# }


# output "sshCommand" {
#   value = "ssh ["${element(azurerm_public_ip.pip.*.fqdn, count.index)}"]@${var.username}"
# }

