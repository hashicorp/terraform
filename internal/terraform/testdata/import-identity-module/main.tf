module "child" {
  source = "./child"
}

import {
  to = aws_lb.foo
  identity = {
    name = "bar"
  }
}

resource "aws_lb" "foo" {}
