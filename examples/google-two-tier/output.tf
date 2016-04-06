output "pool_public_ip" {
  value = "${google_compute_forwarding_rule.default.ip_address}"
}

output "instance_ips" {
  value = "${join(" ", google_compute_instance.www.*.network_interface.0.access_config.0.assigned_nat_ip)}"
}
