resource "null_resource" "foo" {
  count = 0
}

output "test" {
  value = "${sort(null_resource.foo.*.id)}"
}
