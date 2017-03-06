output "hostname" {
  value = "${alicloud_instance.website.instance_name}"
}

output "ecs_id" {
  value = "${alicloud_instance.website.id}"
}