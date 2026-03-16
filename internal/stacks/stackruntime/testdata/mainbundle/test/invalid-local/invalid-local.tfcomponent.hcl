required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

variable "in" {
  type = object({
    name = string
  })
}

locals {
    # This is not caught during the config evaluation but only when we try to
    # evaluate this value during planning / applying.
    invalid_local = { for k, v in var.in : k => v + 3 }
}

component "self" {
  source = "./"
  inputs = {
    name = "example#{local.invalid_local}"
  }
  
  providers = {
    testing = provider.testing.main
  }
}
