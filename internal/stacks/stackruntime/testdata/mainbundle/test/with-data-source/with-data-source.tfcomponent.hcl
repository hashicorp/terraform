required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "id" {
  type = string
}

variable "resource" {
  type = string
}

component "self" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = var.id
    resource = var.resource
  }
}
