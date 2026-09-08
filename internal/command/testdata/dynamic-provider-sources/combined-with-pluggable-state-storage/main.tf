terraform {
  required_providers {
    test = {
      source  = var.provider_source
      version = var.provider_version
    }
  }

  state_store "test_store" {
    provider "test" {}

    value = "foobar"
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

output "greeting" {
  value = "hello ${var.name}"
}
