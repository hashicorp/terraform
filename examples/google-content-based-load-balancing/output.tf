output "application_public_ip" {
  value = "${google_compute_global_forwarding_rule.default.ip_address}"
}
