variable "password" {
  type = string
}

resource "test_resource" "resource" {
  id = "9ddca5a9"
  value = var.password
}

output "password" {
  value = var.password
  sensitive = true
}
