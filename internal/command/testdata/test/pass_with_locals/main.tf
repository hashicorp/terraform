resource "test_resource" "foo" {
  value = "bar"
}

locals {
  value = test_resource.foo.value
}
