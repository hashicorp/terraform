resource "test_resource" "test_id" {
  value = "test"
}

output "test_id" {
  value = test_resource.test_id.id
}
