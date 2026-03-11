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
  
  validation {
    condition     = !contains(["bad", "invalid", "nope"], var.input)
    error_message = "Input cannot be 'bad', 'invalid', or 'nope'."
  }
  
  validation {
    condition     = can(regex("^[A-Z]", var.input))
    error_message = "Input must start with an uppercase letter."
  }
}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
  }
}
