required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "value" {
  type = string
}

component "one" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = "one"
    value = var.value
  }
}

component "two" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = "two"
    value = component.one.value
  }
}

component "three" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = "three"
    value = component.two.value
  }
}

output "value" {
  value = component.three.value
  type = string
}
