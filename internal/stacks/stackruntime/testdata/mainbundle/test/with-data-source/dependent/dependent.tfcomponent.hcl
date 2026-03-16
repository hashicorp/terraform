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

component "self" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = var.id
  }
}

component "data" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = component.self.id
    resource = "resource"
  }
}
