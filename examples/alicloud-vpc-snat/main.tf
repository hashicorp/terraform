provider "alicloud" {
	region = "cn-hangzhou"
}

data "alicloud_instance_types" "1c2g" {
	cpu_core_count = 1
	memory_size = 2
	instance_type_family = "ecs.n1"
}

data "alicloud_zones" "default" {
	"available_instance_type"= "${data.alicloud_instance_types.1c2g.instance_types.0.id}"
	"available_disk_category"= "${var.disk_category}"
}

resource "alicloud_vpc" "default" {
	name = "tf_vpc"
	cidr_block = "${var.vpc_cidr}"
}

resource "alicloud_vswitch" "default" {
	vpc_id = "${alicloud_vpc.default.id}"
	cidr_block = "${var.vswitch_cidr}"
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
}

resource "alicloud_nat_gateway" "default" {
	vpc_id = "${alicloud_vpc.default.id}"
	spec = "Small"
	name = "test_foo"
	bandwidth_packages = [{
		ip_count = 2
		bandwidth = 5
		zone = "${data.alicloud_zones.default.zones.0.id}"
	}]
	depends_on = [
		"alicloud_vswitch.default"]
}
resource "alicloud_snat_entry" "default"{
	snat_table_id = "${alicloud_nat_gateway.default.snat_table_ids}"
	source_vswitch_id = "${alicloud_vswitch.default.id}"
	snat_ip = "${element(split(",", alicloud_nat_gateway.default.bandwidth_packages.0.public_ip_addresses),0)}"
}

resource "alicloud_forward_entry" "default"{
	forward_table_id = "${alicloud_nat_gateway.default.forward_table_ids}"
	external_ip = "${element(split(",", alicloud_nat_gateway.default.bandwidth_packages.0.public_ip_addresses),1)}"
	external_port = "80"
	ip_protocol = "tcp"
	internal_ip = "${alicloud_instance.default.private_ip}"
	internal_port = "8080"
}

resource "alicloud_security_group" "sg" {
	name = "tf_sg"
	description = "tf_sg"
	vpc_id = "${alicloud_vpc.default.id}"
}

resource "alicloud_security_group_rule" "http-in" {
	type = "ingress"
	ip_protocol = "tcp"
	nic_type = "intranet"
	policy = "accept"
	port_range = "80/80"
	priority = 1
	security_group_id = "${alicloud_security_group.sg.id}"
	cidr_ip = "0.0.0.0/0"
}

resource "alicloud_instance" "default" {
	# cn-beijing
	availability_zone = "${data.alicloud_zones.default.zones.0.id}"
	security_groups = ["${alicloud_security_group.sg.id}"]

	vswitch_id = "${alicloud_vswitch.default.id}"

	# series II
	instance_charge_type = "PostPaid"
	instance_type = "${var.instance_type}"
	internet_max_bandwidth_out = 0
	io_optimized = "${var.io_optimized}"

	system_disk_category = "cloud_efficiency"
	image_id = "${var.image_id}"
	instance_name = "tf_vpc_snat"
}