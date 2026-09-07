terraform {
  required_providers {
    simple6 = {
      source = "registry.terraform.io/hashicorp/simple6"
    }
  }

  state_store "simple6_fs" {
    provider "simple6" {}

    workspace_dir = "states"
  }
}

variable "name" {
  default = "world"
}

resource "terraform_data" "my-data" {
  input = "hello ${var.name}"
}
