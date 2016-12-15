output "vpc_id" {
  value = "${alicloud_vpc.main.id}"
}

output "vswitch_ids" {
  value = "${join(",", alicloud_vswitch.main.*.id)}"
}

output "availability_zones" {
  value = "${join(",",alicloud_vswitch.main.*.availability_zone)}"
}
