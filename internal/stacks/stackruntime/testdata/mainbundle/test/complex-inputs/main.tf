terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "input" {
  type = list(object({
    id    = string
    value = string
  }))
}

resource "testing_resource" "data" {
  count = length(var.input)
  id    = var.input[count.index].id
  value = var.input[count.index].value
}
