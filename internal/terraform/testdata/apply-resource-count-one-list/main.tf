resource "null_resource" "foo" {
  count = 1
}

output "test" {
  value = "${sort(null_resource.foo.*.id)}"
}
