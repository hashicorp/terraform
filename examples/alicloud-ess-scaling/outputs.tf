output "scaling_group_id" {
  value = "${alicloud_ess_scaling_group.scaling.id}"
}

output "configuration_id" {
  value = "${alicloud_ess_scaling_configuration.config.id}"
}