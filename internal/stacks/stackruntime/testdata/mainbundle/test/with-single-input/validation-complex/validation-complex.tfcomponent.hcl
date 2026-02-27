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

variable "email" {
  type = string
  
  validation {
    condition     = can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.email))
    error_message = "Must be a valid email address."
  }
}

variable "ip_address" {
  type = string
  
  validation {
    condition     = can(regex("^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$", var.ip_address))
    error_message = "Must be a valid IPv4 address."
  }
}

variable "environment" {
  type = string
  
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}

variable "tags" {
  type = map(string)
  
  validation {
    condition     = alltrue([for k, v in var.tags : can(regex("^[a-z][a-z0-9-]*$", k))])
    error_message = "Tag keys must start with lowercase letter and contain only lowercase letters, numbers, and hyphens."
  }
  
  validation {
    condition     = alltrue([for k, v in var.tags : length(v) > 0 && length(v) <= 256])
    error_message = "Tag values must be 1-256 characters."
  }
  
  validation {
    condition     = contains(keys(var.tags), "owner")
    error_message = "Tags must include 'owner' key."
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
