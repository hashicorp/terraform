resource "testing_deferred_resource" "data" {
  id       = "upstream"
  deferred = true
}

output "id" {
  value = testing_deferred_resource.data.id
}
