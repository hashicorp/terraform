resource "alicloud_vpc" "main" {
  cidr_block = "${var.vpc_cidr}"
}

resource "alicloud_vswitch" "main" {
  vpc_id = "${alicloud_vpc.main.id}"
  cidr_block = "${var.vswitch_cidr}"
  availability_zone = "${var.zone}"
  depends_on = ["alicloud_vpc.main"]
}

resource "alicloud_route_entry" "entry" {
  router_id = "${alicloud_vpc.main.router_id}"
  route_table_id = "${alicloud_vpc.main.router_table_id}"
  destination_cidrblock = "0.0.0.0/0"
  nexthop_type = "Instance"
  nexthop_id = "${alicloud_instance.nat.id}"
}

resource "alicloud_instance" "nat" {
  image_id = "${var.image}"
  instance_type = "${var.instance_nat_type}"
  availability_zone = "${var.zone}"
  security_groups = ["${alicloud_security_group.group.id}"]
  vswitch_id = "${alicloud_vswitch.main.id}"
  instance_name = "nat"
  io_optimized = "optimized"
  system_disk_category = "cloud_efficiency"
  password= "${var.instance_pwd}"

  depends_on = ["alicloud_instance.worker"]
  user_data = "${data.template_file.shell.rendered}"

  tags {
    Name = "ecs-nat"
  }
}

data "template_file" "shell" {
  template = "${file("userdata.sh")}"

  vars {
      worker_private_ip = "${alicloud_instance.worker.private_ip}"
      vswitch_cidr = "${var.vswitch_cidr}"
  }
}

resource "alicloud_instance" "worker" {
  image_id = "${var.image}"
  instance_type = "${var.instance_worker_type}"
  availability_zone = "${var.zone}"
  security_groups = ["${alicloud_security_group.group.id}"]
  vswitch_id = "${alicloud_vswitch.main.id}"
  instance_name = "worker"
  io_optimized = "optimized"
  system_disk_category = "cloud_efficiency"
  password= "${var.instance_pwd}"

  tags {
    Name = "ecs-worker"
  }
}

resource "alicloud_eip" "eip" {
}

resource "alicloud_eip_association" "attach" {
  allocation_id = "${alicloud_eip.eip.id}"
  instance_id = "${alicloud_instance.nat.id}"
}

resource "alicloud_security_group" "group" {
  name = "terraform-test-group"
  description = "New security group"
  vpc_id = "${alicloud_vpc.main.id}"
}

resource "alicloud_security_group_rule" "allow_in" {
  security_group_id = "${alicloud_security_group.group.id}"
  type = "ingress"
  cidr_ip= "0.0.0.0/0"
  policy = "accept"
  ip_protocol= "all"
  nic_type= "intranet"
  port_range= "-1/-1"
  priority= 1
}

resource "alicloud_security_group_rule" "allow_out" {
  security_group_id = "${alicloud_security_group.group.id}"
  type = "egress"
  cidr_ip= "0.0.0.0/0"
  policy = "accept"
  ip_protocol= "all"
  nic_type= "intranet"
  port_range= "-1/-1"
  priority= 1
}