locals {
  val = 2
  m = {
    "a" = "b"
  }
}

module "count_child" {
  count = local.val
  source = "./child"
}

module "for_each_child" {
  for_each = test_object.foo
  source = "./child"
}

resource "test_object" "foo" {
  for_each = local.m
}