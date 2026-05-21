
terraform {
  required_providers {
    test = {
      source = "terraform.io/builtin/test"

      configuration_aliases = [ test ]
    }
  }
}

action "test_action" "notify" {
  config {
    message = "resource created"
  }
}

resource "test_resource" "main" {
  value = "example"

  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [action.test_action.notify]
    }
  }
}

output "result" {
  value = test_resource.main.value
}
