variable "the_id" {
  default = "123"
}

import {
  to = aws_instance.foo
  id = var.the_id
}

resource "aws_instance" "foo" {
}

module "test" {
  source = "./mod"
}

import {
  to = module.test.aws_instance.foo
  id = var.the_id
}

