resource "alicloud_slb" "instance" {
  name = "${var.slb_name}"
  internet_charge_type = "${var.internet_charge_type}"
  internet = "${var.internet}"

  listener = [
    {
      "instance_port" = "2111"
      "lb_port" = "21"
      "lb_protocol" = "tcp"
      "bandwidth" = "5"
    },
    {
      "instance_port" = "8000"
      "lb_port" = "80"
      "lb_protocol" = "http"
      "bandwidth" = "5"
    },
    {
      "instance_port" = "1611"
      "lb_port" = "161"
      "lb_protocol" = "udp"
      "bandwidth" = "5"
    }]
}
