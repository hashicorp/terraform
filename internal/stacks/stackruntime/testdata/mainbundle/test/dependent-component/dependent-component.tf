terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "requirements" {
  type = set(string)
}

variable "id" {
  type     = string
  default  = null
  nullable = true # We'll generate an ID if none provided.
}

resource "testing_blocked_resource" "resource" {
  id = var.id
  required_resources = var.requirements
}
