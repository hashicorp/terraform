resource "test_resource" "foo" {
  value = "from_child"
}

output "value" {
  value = test_resource.foo.value
}
