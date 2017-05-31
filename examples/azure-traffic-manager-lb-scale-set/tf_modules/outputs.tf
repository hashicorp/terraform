# Output public IP ID (Load Balancer) for traffic manager

output "webserverpublic_ip_id" {
  value = "${azurerm_public_ip.webserverpublic_ip.id}"
}