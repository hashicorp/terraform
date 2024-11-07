terraform {
  required_providers {
    terraform = {
      source = "terraform.io/builtin/terraform"
    }
  }
}

resource "terraform_data" "main" {
  input = "hello"
}

output "input" {
  value = terraform_data.main.input
}

output "output" {
  value = terraform_data.main.output
}
