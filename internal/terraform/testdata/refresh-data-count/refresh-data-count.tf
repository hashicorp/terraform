resource "test" "foo" {
}

data "test" "foo" {
  count = length(test.foo.things)
}
