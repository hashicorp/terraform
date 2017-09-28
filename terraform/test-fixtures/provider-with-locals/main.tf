provider "aws" {
	alias = "${local.foo}"
}

locals {
	foo = "bar"
}

resource "aws_instance" "foo" {
    value = "${local.foo}"
}
