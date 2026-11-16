module "child" {
  source = "./child"
}

import {
  to = aws_lb.foo
  id = module.child.lb_id
}

resource "aws_lb" "foo" {}
