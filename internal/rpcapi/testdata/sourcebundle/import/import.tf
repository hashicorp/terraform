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

import {
  id = var.id
  to = testing_resource.resource
}

resource "testing_resource" "resource" {}
