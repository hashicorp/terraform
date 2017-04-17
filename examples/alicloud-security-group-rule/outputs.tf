output "rule_id" {
  value = "${alicloud_security_group_rule.allow_all_tcp.id}"
}

output "rule_type" {
  value = "${alicloud_security_group_rule.allow_all_tcp.type}"
}

output "port_range" {
  value = "${alicloud_security_group_rule.allow_all_tcp.port_range}"
}

output "ip_protocol" {
  value = "${alicloud_security_group_rule.allow_all_tcp.ip_protocol}"
}