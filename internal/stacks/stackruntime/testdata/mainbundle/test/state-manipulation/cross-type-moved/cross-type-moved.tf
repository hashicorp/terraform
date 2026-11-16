terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

moved {
  from = testing_resource.before
  to   = testing_deferred_resource.after
}

resource "testing_deferred_resource" "after" {
  id = "moved"
  value = "moved"
  deferred = false
}
