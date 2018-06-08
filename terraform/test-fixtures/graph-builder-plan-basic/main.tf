variable "foo" {
    default = "bar"
    description = "bar"
}

provider "aws" {
    foo = "${openstack_floating_ip.random.value}"
}

resource "openstack_floating_ip" "random" {}

resource "aws_security_group" "firewall" {}

resource "aws_instance" "web" {
    ami = "${var.foo}"
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]
}

resource "aws_load_balancer" "weblb" {
    members = "${aws_instance.web.id_list}"
}

locals {
  instance_id = "${aws_instance.web.id}"
}

output "instance_id" {
  value = "${local.instance_id}"
}
