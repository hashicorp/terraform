terraform {
  required_providers {
    terraform = {
      source = "terraform.io/builtin/terraform"
    }
  }
}

variable "input" {
  type = string
}

resource "terraform_data" "main" {
  input = var.input
}

output "output" {
  value = terraform_data.main.output
}
