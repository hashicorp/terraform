variable "foo" {
  default     = "bar"
  description = "bar"
}

provider "aws" {
  test_string = "${openstack_floating_ip.random.test_string}"
}

resource "openstack_floating_ip" "random" {}

resource "aws_security_group" "firewall" {}

resource "aws_instance" "web" {
  test_string = var.foo

  test_list = [
    "foo",
    aws_security_group.firewall.test_string,
  ]
}

resource "aws_load_balancer" "weblb" {
  test_list = aws_instance.web.test_list
}

locals {
  instance_id = "${aws_instance.web.test_string}"
}

output "instance_id" {
  value = "${local.instance_id}"
}
