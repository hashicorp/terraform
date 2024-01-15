
resource "test_resource" "resource" {}

output "id" {
  value = test_resource.resource.id
}
