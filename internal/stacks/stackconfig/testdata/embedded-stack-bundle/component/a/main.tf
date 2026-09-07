variable "name" {
  type = string
}

resource "null_resource" "example" {
  triggers = {
    name = var.name
  }
}

output "greeting" {
  value = "Hello, ${var.name}!"
}

output "resource_id" {
  value = null_resource.example.id
}
