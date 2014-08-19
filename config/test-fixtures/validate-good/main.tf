variable "foo" {
    default = "bar"
    description = "bar"
}

variable "amis" {
    default = {
        east = "foo"
    }
}

provider "aws" {
  access_key = "foo"
  secret_key = "bar"
}

provider "do" {
  api_key = "${var.foo}"
}

resource "aws_security_group" "firewall" {
}

resource aws_instance "web" {
    ami = "${var.amis.east}"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]

    network_interface {
        device_index = 0
        description = "Main network interface"
    }

    depends_on = ["aws_security_group.firewall"]
}
