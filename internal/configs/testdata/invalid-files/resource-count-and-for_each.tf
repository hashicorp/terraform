resource "test" "foo" {
  count = 2
  for_each = ["a"]
}
