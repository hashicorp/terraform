terraform {
    required_version = "foo"
}

variable "foo" {
    default = "bar"
    description = "bar"
}

variable "bar" {
    type = "string"
}

variable "baz" {
    type = "map"

    default = {
        key = "value"
    }
}

provider "aws" {
  access_key = "foo"
  secret_key = "bar"
}

provider "do" {
  api_key = "${var.foo}"
}

data "do" "simple" {
    foo = "baz"
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

    provisioner "file" {
        source = "foo"
        destination = "bar"
    }
}

locals {
  security_group_ids = "${aws_security_group.firewall.*.id}"
  web_ip = "${aws_instance.web.private_ip}"
}

locals {
  literal = 2
  literal_list = ["foo"]
  literal_map = {"foo" = "bar"}
}

resource "aws_instance" "db" {
    security_groups = "${aws_security_group.firewall.*.id}"
    VPC = "foo"

    depends_on = ["aws_instance.web"]

    provisioner "file" {
        source = "foo"
        destination = "bar"
    }
}

output "web_ip" {
    value = "${aws_instance.web.private_ip}"
}

output "web_id" {
    description = "The ID"
    value = "${aws_instance.web.id}"
}

atlas {
    name = "mitchellh/foo"
}
