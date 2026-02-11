required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = string
  
  validation {
    condition     = length(var.input) > 5
    error_message = "Input must be longer than 5 characters."
  }
  
  validation {
    condition     = startswith(var.input, "H")
    error_message = "Input must start with H."
  }
}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = var.id
  }
}
