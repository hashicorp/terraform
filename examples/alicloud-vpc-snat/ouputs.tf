output "instance_id" {
  value = "${alicloud_instance.default.id}"
}

output "bindwidth_package_ip" {
  value = "${alicloud_nat_gateway.default.bandwidth_packages.0.public_ip_addresses}"
}
