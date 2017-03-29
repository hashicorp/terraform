output "nat_instance_id" {
  value = "${alicloud_instance.nat.id}"
}

output "nat_instance_private_ip" {
  value = "${alicloud_instance.nat.private_ip}"
}

output "nat_instance_eip_address" {
  value = "${alicloud_eip.eip.ip_address}"
}

output "worker_instance_id" {
  value = "${alicloud_instance.worker.id}"
}

output "worker_instance_private_ip" {
  value = "${alicloud_instance.worker.private_ip}"
}