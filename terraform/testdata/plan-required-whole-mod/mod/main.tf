resource "test_resource" "for_output" {
  required = "val"
}

output "object" {
  value = test_resource.for_output
}
