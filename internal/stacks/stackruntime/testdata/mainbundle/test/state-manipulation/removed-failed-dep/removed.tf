terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

resource "testing_failed_resource" "resource" {
  fail_apply = true
}

removed {
  from = testing_resource.resource

  lifecycle {
    destroy = false
  }
}
