provider "alicloud" {
  alias = "bj"
  region = "${var.region1}"
}

provider "alicloud" {
  alias = "hz"
  region = "${var.region2}"
}

resource "alicloud_vpc" "work" {
  provider = "alicloud.hz"
  name = "${var.long_name}"
  cidr_block = "${var.vpc_cidr}"
}

resource "alicloud_vpc" "control" {
  provider = "alicloud.bj"
  name = "${var.long_name}"
  cidr_block = "${var.vpc_cidr}"
}
