terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

resource "testing_resource" "this" {
  id    = "test"
  value = "hello"
  
  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.testing_action.example]
    }
  }
}

action "testing_action" "example" {
  config {
    message = "Test action invocation"
  }
}
