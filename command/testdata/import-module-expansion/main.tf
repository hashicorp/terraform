locals {
  xs = toset(["foo"])
}

module "a" {
  for_each = local.xs
  source   = "./a"
}

module "b" {
  for_each = local.xs
  source   = "./b"
  y = module.a[each.key].y
}

resource "test_instance" "foo" {}
