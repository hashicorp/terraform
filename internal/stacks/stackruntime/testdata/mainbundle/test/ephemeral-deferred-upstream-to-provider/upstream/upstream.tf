resource "testing_deferred_resource" "data" {
  id       = "upstream"
  deferred = true
}

output "static" {
  value = "known"
}
