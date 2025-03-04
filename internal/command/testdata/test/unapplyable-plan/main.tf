
resource "test_resource" "example" {
  value = "bar"
}

output "value" {
  value = test_resource.example.value
}
