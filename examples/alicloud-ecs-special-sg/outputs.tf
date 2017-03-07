output "hostname_list" {
  value = "${join(",", alicloud_instance.instance.*.instance_name)}"
}

output "ecs_ids" {
  value = "${join(",", alicloud_instance.instance.*.id)}"
}
