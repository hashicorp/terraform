resource "alicloud_slb" "instance" {
  name = "${var.slb_name}"
  internet_charge_type = "${var.internet_charge_type}"
  internet = "${var.internet}"

  listener = [
    {
      "instance_port" = "2375"
      "instance_protocol" = "tcp"
      "lb_port" = "3376"
      "lb_protocol" = "tcp"
      "bandwidth" = "5"
    }]
}
