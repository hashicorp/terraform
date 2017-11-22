resource "test_instance" "foo" {
}

output "foo_id" {
  value = "${test_instance.foo.id}"
}
