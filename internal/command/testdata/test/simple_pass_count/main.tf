resource "test_resource" "foo" {
  count = 3
  value = "bar"
}
