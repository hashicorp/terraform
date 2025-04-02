terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "datasource_id" {
  type = string
}

variable "resource_id" {
  type = string
}

variable "write_only_input" {
  type = string
  sensitive = true
}

data "testing_write_only_data_source" "data" {
  id = var.datasource_id
  write_only = var.write_only_input
}

resource "testing_write_only_resource" "data" {
  id    = var.resource_id
  value = data.testing_write_only_data_source.data.value
  write_only = var.write_only_input
}
