output "slb_id" {
  value = "${alicloud_slb.instance.id}"
}

output "slbname" {
  value = "${alicloud_slb.instance.name}"
}

output "instances" {
  value = "${alicloud_slb.instance.instances}"
}