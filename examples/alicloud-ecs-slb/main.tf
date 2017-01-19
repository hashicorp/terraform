resource "alicloud_security_group" "group" {
  name = "${var.short_name}"
  description = "New security group"
}

resource "alicloud_instance" "instance" {
  instance_name = "${var.short_name}-${var.role}-${format(var.count_format, count.index+1)}"
  host_name = "${var.short_name}-${var.role}-${format(var.count_format, count.index+1)}"
  image_id = "${var.image_id}"
  instance_type = "${var.ecs_type}"
  count = "${var.count}"
  availability_zone = "${var.availability_zones}"
  security_groups = ["${alicloud_security_group.group.*.id}"]

  internet_charge_type = "${var.internet_charge_type}"
  internet_max_bandwidth_out = "${var.internet_max_bandwidth_out}"

  io_optimized = "${var.io_optimized}"

  password = "${var.ecs_password}"

  allocate_public_ip = "${var.allocate_public_ip}"

  instance_charge_type = "PostPaid"
  system_disk_category = "cloud_efficiency"


  tags {
    role = "${var.role}"
    dc = "${var.datacenter}"
  }

}

resource "alicloud_slb" "instance" {
  name = "${var.slb_name}"
  internet_charge_type = "${var.slb_internet_charge_type}"
  internet = "${var.internet}"

  listener = [
    {
      "instance_port" = "2111"
      "lb_port" = "21"
      "lb_protocol" = "tcp"
      "bandwidth" = "5"
    }]
}


resource "alicloud_slb_attachment" "default" {
  slb_id = "${alicloud_slb.instance.id}"
  instances = ["${alicloud_instance.instance.*.id}"]
}