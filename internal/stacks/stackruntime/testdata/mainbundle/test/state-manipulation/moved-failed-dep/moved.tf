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

moved {
  from = testing_resource.before
  to   = testing_resource.after
}

resource "testing_resource" "after" {
  id = "moved"
  value = "moved"

  depends_on = [testing_failed_resource.resource]
}
