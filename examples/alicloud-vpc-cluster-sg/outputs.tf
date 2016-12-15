output "default_security_group" {
  value = "${alicloud_security_group.default.id}"
}

output "edge_security_group" {
  value = "${alicloud_security_group.edge.id}"
}

output "control_security_group" {
  value = "${alicloud_security_group.control.id}"
}

output "worker_security_group" {
  value = "${alicloud_security_group.worker.id}"
}
