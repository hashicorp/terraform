#terraform:hcl2

terraform {
  required_version = "foo"

  backend "baz" {
    something = "nothing"
  }
}

variable "foo" {
  default     = "bar"
  description = "barbar"
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
  version    = "1.0.0"
}

provider "do" {
  api_key = var.foo
  alias   = "fum"
}

data "do" "simple" {
  foo      = "baz"
  provider = "do.foo"
}

data "do" "depends" {
  depends_on = ["data.do.simple"]
}

resource "aws_security_group" "firewall" {
  count    = 5
  provider = "another"
}

resource "aws_instance" "web" {
  ami = "${var.foo}"
  security_groups = [
    "foo",
    aws_security_group.firewall.foo,
  ]

  network_interface {
    device_index = 0
    description  = "Main network interface"
  }

  connection {
    default = true
  }

  provisioner "file" {
    source      = "foo"
    destination = "bar"
  }
}

locals {
  security_group_ids = aws_security_group.firewall.*.id
  web_ip = aws_instance.web.private_ip
}

locals {
  literal = 2
  literal_list = ["foo"]
  literal_map = {"foo" = "bar"}
}

resource "aws_instance" "db" {
  security_groups = aws_security_group.firewall.*.id
  VPC             = "foo"

  tags = {
    Name = "${var.bar}-database"
  }

  depends_on = ["aws_instance.web"]

  provisioner "file" {
    source      = "here"
    destination = "there"

    connection {
      default = false
    }
  }
}

output "web_ip" {
  value     = aws_instance.web.private_ip
  sensitive = true
}

output "web_id" {
  description = "The ID"
  value       = aws_instance.web.id
  depends_on  = ["aws_instance.db"]
}

atlas {
  name = "example/foo"
}

module "child" {
  source = "./baz"

  toasty = true
}
