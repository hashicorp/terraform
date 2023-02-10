resource "foo" "bar" {
}

locals {
  foo_bar_baz = foo.bar.baz
}

resource "foo" "baz" {
  arg = local.foo_bar_baz
}

module "child" {
  source = "./child"

  in = local.foo_bar_baz
}

resource "foo" "boop" {
  arg = module.child.out
}
