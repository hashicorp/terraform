data "testing_data_source" "credentials" {
  id = "credentials"
}

output "credentials" {
  value = data.testing_data_source.credentials.value
  sensitive = true
}
