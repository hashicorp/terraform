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

variable "number_input" {
  type = number
  
  validation {
    condition     = var.number_input > 0 && var.number_input < 100
    error_message = "Number must be between 0 and 100."
  }
}

variable "list_input" {
  type = list(string)
  
  validation {
    condition     = length(var.list_input) > 0 && length(var.list_input) <= 5
    error_message = "List must contain 1-5 items."
  }
  
  validation {
    condition     = alltrue([for s in var.list_input : length(s) > 0])
    error_message = "List items cannot be empty strings."
  }
}

variable "map_input" {
  type = map(string)
  
  validation {
    condition     = contains(keys(var.map_input), "required_key")
    error_message = "Map must contain 'required_key'."
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
