terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

module "child" {
  source = "./child"
}

output "value" {
  value     = module.child.value
  ephemeral = true
}
