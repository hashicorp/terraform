resource "alicloud_disk" "disk" {
  availability_zone = "${var.availability_zones}"
  category = "${var.disk_category}"
  size = "${var.disk_size}"
  count = "${var.count}"
}

resource "alicloud_instance" "instance" {
  instance_name = "${var.short_name}-${var.role}-${format(var.count_format, count.index+1)}"
  host_name = "${var.short_name}-${var.role}-${format(var.count_format, count.index+1)}"
  image_id = "${var.image_id}"
  instance_type = "${var.ecs_type}"
  count = "${var.count}"
  availability_zone = "${var.availability_zones}"
  security_groups = ["${var.security_groups}"]
  vswitch_id = "${var.vswitch_id}"

  internet_charge_type = "${var.internet_charge_type}"
  internet_max_bandwidth_out = "${var.internet_max_bandwidth_out}"

  io_optimized = "${var.io_optimized}"

  allocate_public_ip = "${var.allocate_public_ip}"

  password = "${var.ecs_password}"

  instance_charge_type = "${var.instance_charge_type}"
  system_disk_category = "${var.system_disk_category}"


  tags {
    role = "${var.role}"
    dc = "${var.datacenter}"
  }

}

resource "alicloud_disk_attachment" "instance-attachment" {
  count = "${var.count}"
  disk_id = "${element(alicloud_disk.disk.*.id, count.index)}"
  instance_id = "${element(alicloud_instance.instance.*.id, count.index)}"
  device_name = "${var.device_name}"
}


