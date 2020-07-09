variable "foo" {}

locals {
  baz = "baz-${var.foo}"
}

provider "aws" {
  foo = "${local.baz}"
}

resource "aws_instance" "foo" {
  id = "bar"
}
