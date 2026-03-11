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

variable "password" {
  type      = string
  sensitive = true
  
  validation {
    condition     = length(var.password) >= 8
    error_message = "Password must be at least 8 characters long."
  }
  
  validation {
    condition     = can(regex("[A-Z]", var.password))
    error_message = "Password must contain at least one uppercase letter."
  }
  
  validation {
    condition     = can(regex("[0-9]", var.password))
    error_message = "Password must contain at least one number."
  }
}

variable "api_key" {
  type      = string
  sensitive = true
  
  validation {
    condition     = length(var.api_key) == 32
    error_message = "API key must be exactly 32 characters."
  }
  
  validation {
    condition     = can(regex("^[a-f0-9]+$", var.api_key))
    error_message = "API key must only contain lowercase hex characters."
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
