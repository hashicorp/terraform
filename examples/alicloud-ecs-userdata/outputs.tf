
output "ecs_id" {
  value = "${alicloud_instance.website.id}"
}

output "ecs_public_ip" {
  value = "${alicloud_instance.website.public_ip}"
}