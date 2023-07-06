resource "test_resource" "foo" {
  value = "bar"
}

output "value" {
  value = test_resource.foo.value
}
