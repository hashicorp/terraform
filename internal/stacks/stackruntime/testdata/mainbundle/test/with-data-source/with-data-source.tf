terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "id" {
  type = string
}

variable "resource" {
  type = string
}

data "testing_data_source" "data" {
  id = var.id
}

resource "testing_resource" "data" {
  id    = var.resource
  value = data.testing_data_source.data.value
}
