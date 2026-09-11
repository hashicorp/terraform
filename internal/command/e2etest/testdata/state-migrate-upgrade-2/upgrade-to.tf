terraform {
  required_providers {
    simple6 = {
      source  = "registry.terraform.io/hashicorp/simple6"
      version = "2.0.0"
    }
  }

  state_store "simple6_fs" {
    provider "simple6" {}

    workspace_dir = "v2.tfstate.d"
    attr_v2       = "foobar" # Attribute name set as 'attr_v2' during build
  }
}

variable "name" {
  default = "world"
}

resource "terraform_data" "my-data" {
  input = "hello ${var.name}"
}
