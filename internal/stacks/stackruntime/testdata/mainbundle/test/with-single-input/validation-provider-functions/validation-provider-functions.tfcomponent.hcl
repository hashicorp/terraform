required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "input" {
  type = string
  default = "default"
}

# Test case: Validation using provider-defined function in condition
variable "echo_value" {
  type = string
  
  validation {
    condition     = provider::testing::echo(var.echo_value) == var.echo_value
    error_message = "Echo function did not return the same value."
  }
}

# Test case: Validation using provider function with built-in functions
variable "combined" {
  type = string
  
  validation {
    condition     = length(provider::testing::echo(var.combined)) > 5
    error_message = "Combined value must be longer than 5 characters after echo."
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

provider "testing" "default" {}
