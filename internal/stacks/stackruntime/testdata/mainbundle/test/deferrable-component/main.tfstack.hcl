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

variable "defer" {
  type = bool
}

component "self" {
  source = "./"


  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = var.id
    defer = var.defer
  }

}