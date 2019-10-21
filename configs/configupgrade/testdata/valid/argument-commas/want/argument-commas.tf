locals {
  foo = "bar"
  baz = "boop"
}

resource "test_instance" "foo" {
  image = "b"
  type  = "d"
}
