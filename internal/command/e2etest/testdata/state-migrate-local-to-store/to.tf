terraform {
  required_providers {
    simple6 = {
      source  = "registry.terraform.io/hashicorp/simple6"
      version = "1.0.0"
    }
  }
  state_store "simple6_fs" {
    provider "simple6" {}
    workspace_dir = "new-location"
  }
}

variable "name" {
  default = "world"
}

resource "terraform_data" "my-data" {
  input = "hello ${var.name}"
}
