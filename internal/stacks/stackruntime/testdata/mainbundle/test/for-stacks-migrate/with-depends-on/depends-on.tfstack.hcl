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

variable "empty" {
  type    = set(string)
  default = []
}

component "empty" {
  source = "./"

  for_each = var.empty

  providers = {
      testing = provider.testing.default
  }

  inputs = {
      input = var.input
  }
}

component "first" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
  }
}

component "second" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
  }

  depends_on = [component.first, stack.embedded]
}

stack "embedded" {
  source = "./valid"

  inputs = {
    input = var.input
  }
}

# stack "second" {
#   source = "./valid"

#   inputs = {
#     input = var.input
#   }
# }
