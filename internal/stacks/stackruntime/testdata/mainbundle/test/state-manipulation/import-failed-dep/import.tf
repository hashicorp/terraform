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

resource "testing_failed_resource" "resource" {
  fail_apply = true
}

import {
  to = testing_resource.data
  id = var.id
}

resource "testing_resource" "data" {
  id = var.id
  value = "imported"

  depends_on = [testing_failed_resource.resource]
}
