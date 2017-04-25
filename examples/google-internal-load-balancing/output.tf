output "internal_load_balancer_ip" {
  value = "${google_compute_forwarding_rule.my-int-lb-forwarding-rule.ip_address}"
}
