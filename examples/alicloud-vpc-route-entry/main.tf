resource "alicloud_vpc" "default" {
	name = "tf_vpc"
	cidr_block = "${var.vpc_cidr}"
}

resource "alicloud_vswitch" "default" {
	vpc_id = "${alicloud_vpc.default.id}"
	cidr_block = "${var.vswitch_cidr}"
	availability_zone = "${var.zone_id}"
}

resource "alicloud_route_entry" "default" {
	router_id = "${alicloud_vpc.default.router_id}"
	route_table_id = "${alicloud_vpc.default.router_table_id}"
	destination_cidrblock = "${var.entry_cidr}"
	nexthop_type = "Instance"
	nexthop_id = "${alicloud_instance.snat.id}"
}

resource "alicloud_security_group" "sg" {
	name = "tf_sg"
	description = "tf_sg"
	vpc_id = "${alicloud_vpc.default.id}"
}

resource "alicloud_security_group_rule" "ssh-in" {
	type = "ingress"
    ip_protocol = "tcp"
	nic_type = "intranet"
	policy = "${var.rule_policy}"
	port_range = "22/22"
	priority = 1
	security_group_id = "${alicloud_security_group.sg.id}"
	cidr_ip = "0.0.0.0/0"
}

resource "alicloud_security_group_rule" "http-in" {
  type = "ingress"
  ip_protocol = "tcp"
  nic_type = "internet"
  policy = "accept"
  port_range = "80/80"
  priority = 1
  security_group_id = "${alicloud_security_group.sg.id}"
  cidr_ip = "0.0.0.0/0"
}

resource "alicloud_security_group_rule" "https-in" {
  type = "ingress"
  ip_protocol = "tcp"
  nic_type = "internet"
  policy = "accept"
  port_range = "443/443"
  priority = 1
  security_group_id = "${alicloud_security_group.sg.id}"
  cidr_ip = "0.0.0.0/0"
}

resource "alicloud_instance" "snat" {
	# cn-beijing
	availability_zone = "${var.zone_id}"
	security_groups = ["${alicloud_security_group.sg.id}"]

	vswitch_id = "${alicloud_vswitch.default.id}"
	allocate_public_ip = true

	# series II
	instance_charge_type = "PostPaid"
	instance_type = "${var.instance_type}"
	internet_charge_type = "${var.internet_charge_type}"
	internet_max_bandwidth_out = 5
	io_optimized = "${var.io_optimized}"

	system_disk_category = "cloud_efficiency"
	image_id = "${var.image_id}"
	instance_name = "tf_snat"
}