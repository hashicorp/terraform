resource "null_resource" "test" {
}

output "output" {
  value = "${null_resource.test.id}"
}
