terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "id" {
  type     = string
  default  = null
  nullable = true # We'll generate an ID if none provided.
}

variable "input" {
  type = string
  default = null
  nullable = true
}

variable "fail_plan" {
  type = bool
  default = null
  nullable = true
}

variable "fail_apply" {
  type = bool
  default = null
  nullable = true
}

resource "testing_failed_resource" "data" {
  id    = var.id
  value = var.input
  fail_plan = var.fail_plan
  fail_apply = var.fail_apply
}

output "value" {
  value = testing_failed_resource.data.value
}
