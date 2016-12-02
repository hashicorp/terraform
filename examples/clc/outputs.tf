output "group_id" {
  value = "${clc_group.frontends.id}"
}

output "node_id" {
  value = "${clc_server.node.id}"
}

output "node_ip" {
  value = "${clc_server.node.private_ip_address}"
}

output "node_password" {
  value = "${clc_server.node.password}"
}

output "backdoor" {
  value = "${clc_public_ip.backdoor.id}"
}

output "frontdoor" {
  value = "${clc_load_balancer.frontdoor.ip_address}"
}

output "pool" {
  value = "curl -vv ${clc_load_balancer.frontdoor.ip_address}"
}
