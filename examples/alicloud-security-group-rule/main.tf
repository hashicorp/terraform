resource "alicloud_security_group" "default" {
  name = "${var.security_group_name}"
}

resource "alicloud_security_group_rule" "allow_all_tcp" {
  type = "ingress"
  ip_protocol = "tcp"
  nic_type = "${var.nic_type}"
  policy = "accept"
  port_range = "1/65535"
  priority = 1
  security_group_id = "${alicloud_security_group.default.id}"
  cidr_ip = "0.0.0.0/0"
}