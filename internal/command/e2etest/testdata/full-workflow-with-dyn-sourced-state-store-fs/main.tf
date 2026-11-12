terraform {
  required_providers {
    simple6 = {
      source  = var.provider_source
      version = var.provider_version
    }
  }

  state_store "simple6_fs" {
    provider "simple6" {}

    workspace_dir = "states"
  }
}

variable "provider_source" {
  default = "invalid source string" // If the default value is used it will cause an error
  type    = string
  const   = true
}

variable "provider_version" {
  default = "invalid version string" // If the default value is used it will cause an error
  type    = string
  const   = true
}

variable "name" {
  default = "world"
}

resource "terraform_data" "my-data" {
  input = "hello ${var.name}"
}

output "greeting" {
  value = resource.terraform_data.my-data.output
}
