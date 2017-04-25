provider "alicloud" {
  alias = "bj"
  region = "cn-beijing"
}

resource "alicloud_instance" "instance" {
  provider = "alicloud.bj"
  instance_name = "website-${format(var.count_format, count.index+1)}"
  host_name = "website-${format(var.count_format, count.index+1)}"
  image_id = "centos7u2_64_40G_cloudinit_20160728.raw"
  instance_type = "ecs.s2.large"
  count = "6"
  availability_zone = "cn-beijing-b"
  security_groups = "${var.security_groups}"

  internet_charge_type = "PayByBandwidth"

  io_optimized = "none"

  password = "${var.ecs_password}"

  allocate_public_ip = "false"

  instance_charge_type = "PostPaid"
  system_disk_category = "cloud"


  tags {
    env = "prod"
    product = "website"
    dc = "beijing"
  }

}

resource "alicloud_slb_attachment" "foo" {
  slb_id = "${var.slb_id}"
  instances = ["${alicloud_instance.instance.0.id}", "${alicloud_instance.instance.1.id}"]
}