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

# Test case: error_message references sensitive value
variable "password" {
  type      = string
  sensitive = true
  
  validation {
    condition     = length(var.password) >= 8
    error_message = "Password '${var.password}' is too short."
  }
}

# Test case: error_message references ephemeral value
variable "token" {
  type      = string
  ephemeral = true
  
  validation {
    condition     = length(var.token) == 32
    error_message = "Token '${var.token}' is invalid."
  }
}

# Test case: error_message that is not a string
variable "count_value" {
  type = number
  
  validation {
    condition     = var.count_value > 0
    error_message = var.count_value  # Invalid: should be a string
  }
}

# Test case: error_message references sensitive value even when validation passes
variable "api_key" {
  type      = string
  sensitive = true
  
  validation {
    condition     = length(var.api_key) >= 16
    error_message = "API key '${var.api_key}' must be at least 16 characters."
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
