terraform {
  required_providers {
    simple6 = {
      source = "registry.terraform.io/hashicorp/simple6"
    }
  }

  state_store "simple6_inmem" {
    provider "simple6" {}
  }
}

variable "name" {
  default = "world"
}

data "terraform_data" "my-data" {
  input = "hello ${var.name}"
}

output "greeting" {
  value = data.terraform_data.output
}
