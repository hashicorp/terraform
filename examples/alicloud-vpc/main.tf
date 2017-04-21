resource "alicloud_vpc" "main" {
  name = "${var.long_name}"
  cidr_block = "${var.vpc_cidr}"
}

resource "alicloud_vswitch" "main" {
  vpc_id = "${alicloud_vpc.main.id}"
  count = "${length(split(",", var.availability_zones))}"
  cidr_block = "${lookup(var.cidr_blocks, "az${count.index}")}"
  availability_zone = "${var.availability_zones}"
  depends_on = [
    "alicloud_vpc.main"]
}

resource "alicloud_nat_gateway" "main" {
  vpc_id = "${alicloud_vpc.main.id}"
  spec = "Small"
  bandwidth_packages = [
    {
      ip_count = 1
      bandwidth = 5
      zone = "${var.availability_zones}"
    }
  ]
  depends_on = [
    "alicloud_vswitch.main"]
}

