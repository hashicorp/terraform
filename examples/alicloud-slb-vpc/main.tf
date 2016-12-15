resource "alicloud_slb" "instance" {
  name = "${var.name}"
  vpc_id = "${var.vpc_id}"
  vswitch_id = "${var.vswitch_id}"
  instances = "${var.instances}"
  internet_charge_type = "${var.internet_charge_type}"
  internet = "${var.internet}"
  listener = [
    {
      "instance_port" = "3375"
      "instance_protocol" = "tcp"
      "lb_port" = "3376"
      "lb_protocol" = "tcp"
      "bandwidth" = "5"
    }]
}

