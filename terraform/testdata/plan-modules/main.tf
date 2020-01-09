locals {
  val = 2
}

module "child" {
  count = local.val
  source = "./child"
}

output "out" {
  value = module.child[*].out
}

resource "aws_instance" "foo" {
  num = 2
}

resource "aws_instance" "bar" {
  foo = "${aws_instance.foo.num}"
}
