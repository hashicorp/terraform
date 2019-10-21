provider "do" {
  api_key = "${var.foo}"
}

data "do" "depends" {
  depends_on = ["data.do.simple"]
}

resource "aws_security_group" "firewall" {
    count = 5
}

resource aws_instance "web" {
    ami = "${var.foo}"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]

    network_interface {
        device_index = 0
        description = "Main network interface"
    }
}
