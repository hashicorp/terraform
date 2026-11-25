resource "testing_resource" "a" {
  id    = "a"
  value = "a-value"
}

output "value" {
  value = testing_resource.a.value
}
