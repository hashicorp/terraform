resource "test" "foo" {
  things = ["foo"]
}

data "test" "foo" {
  count = length(test.foo.things)
}
