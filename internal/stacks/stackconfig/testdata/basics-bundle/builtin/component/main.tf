variable "name" {
  type = string
}

resource "terraform_data" "example" {
  input = {
    message = "Hello, ${var.name}!"
  }
}

output "greeting" {
  value = terraform_data.example.input.message
}
