required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "default" {
  type = object({
    id    = string
    value = string
  })
  default = {
    id    = "cec9bc39"
    value = "hello, mercury!"
  }
}

variable "optional_default" {
  type = object({
    id    = optional(string)
    value = optional(string, "hello, venus!")
  })
  default = {
    id = "78d8b3d7"
  }
}

variable "optional" {
  type = object({
    id    = optional(string)
    value = optional(string, "hello, earth!")
  })
}

component "parent" {
  source = "../"
  providers = {
    testing = provider.testing.default
  }
  inputs = {
    input = [
      var.default,
      var.optional_default,
      var.optional,
    ]
  }
}
