
variable "name" {
  default = "world"
}

resource "null_resource" "test" {
  triggers = {
    greeting = "Hello ${var.name}"
  }
}

output "greeting" {
  value = null_resource.test.triggers["greeting"]
}
