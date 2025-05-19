required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = string
}

stack "first" {
  source = "../valid"

  inputs = {
    input = var.input
  }

  # nu-uh, this isn't valid.
  depends_on = [var.input, component.missing]
}

component "first" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
  }

  # nu-uh, this isn't valid.
  depends_on = [var.input, stack.missing]
}

component "second" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
  }

  # nu-uh, this isn't valid.
  depends_on = [component.first[1]]
}