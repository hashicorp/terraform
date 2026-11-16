variable "external_file" {
}

resource "test_resource" "test" {
  value = filebase64(var.external_file)
}
