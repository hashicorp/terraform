
resource "alicloud_vpc" "default" {
  name = "tf-vpc"
  cidr_block = "${var.vpc_cidr}"
}

resource "alicloud_vswitch" "vsw" {
  vpc_id = "${alicloud_vpc.default.id}"
  cidr_block = "${var.vswitch_cidr}"
  availability_zone = "${var.zone}"
}

resource "alicloud_security_group" "sg" {
	name = "tf-sg"
	description = "sg"
	vpc_id = "${alicloud_vpc.default.id}"
}

resource "alicloud_instance" "website" {
	# cn-beijing
	availability_zone = "${var.zone}"
	vswitch_id = "${alicloud_vswitch.vsw.id}"
	image_id = "${var.image}"

	# series II
	instance_type = "${var.ecs_type}"
	io_optimized = "optimized"
	system_disk_category = "cloud_efficiency"

	internet_charge_type = "PayByTraffic"
	internet_max_bandwidth_out = 5
	allocate_public_ip = true
	security_groups = ["${alicloud_security_group.sg.id}"]
	instance_name = "test_foo"

	user_data = "${file("userdata.sh")}"
}
