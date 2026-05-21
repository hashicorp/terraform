
terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "id" {
  type     = string
  default  = null
  nullable = true # We'll generate an ID if none provided.
}

variable "defer" {
  type = bool
}

# Action that should be invoked when resource is created
action "testing_action" "notify" {
  config {
    message = "resource created with id ${var.id}"
  }
}

# Deferred resource with action trigger
resource "testing_deferred_resource" "data" {
  id       = var.id
  deferred = var.defer

  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.testing_action.notify]
    }
  }
}
