terraform {
  required_providers {
    test = {
      source = "registry.terraform.io/hashicorp/test"
    }
  }

  state_store "test_store" {
    provider "test" {}

    value = "foobar"
  }
}

variable "name" {
  default = "world"
}

resource "test_instance" "my-data" {
  input = "hello ${var.name}"
}

output "greeting" {
  value = resource.terraform_data.my-data.output
}
