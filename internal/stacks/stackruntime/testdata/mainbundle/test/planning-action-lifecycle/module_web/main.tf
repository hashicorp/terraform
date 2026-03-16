
terraform {
  required_providers {
    testing = {
      source = "terraform.io/builtin/testing"

      configuration_aliases = [ testing ]
    }
  }
}

action "testing_action" "notify" {
  config {
    message = "resource created"
  }
}

resource "testing_resource" "main" {
  value = "example"

  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.testing_action.notify]
    }
  }
}

output "result" {
  value = testing_resource.main.value
}
