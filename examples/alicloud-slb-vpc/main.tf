resource "alicloud_vpc" "main" {
  name = "${var.long_name}"
  cidr_block = "${var.vpc_cidr}"
}

resource "alicloud_vswitch" "main" {
  vpc_id = "${alicloud_vpc.main.id}"
  count = "${length(split(",", var.availability_zones))}"
  cidr_block = "${lookup(var.cidr_blocks, "az${count.index}")}"
  availability_zone = "${element(split(",", var.availability_zones), count.index)}"
  depends_on = [
    "alicloud_vpc.main"]
}

resource "alicloud_slb" "instance" {
  name = "${var.name}"
  vswitch_id = "${alicloud_vswitch.main.id}"
  internet_charge_type = "${var.internet_charge_type}"
  listener = [
    {
      "instance_port" = "2111"
      "lb_port" = "21"
      "lb_protocol" = "tcp"
      "bandwidth" = "5"
    }]
}

