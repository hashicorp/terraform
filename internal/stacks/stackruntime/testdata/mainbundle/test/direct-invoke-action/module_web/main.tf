
terraform {
  required_providers {
    testing = {
      source = "terraform.io/builtin/testing"

      configuration_aliases = [testing]
    }
  }
}

# A standalone, directly-invocable action. It is not wired to any resource
# lifecycle; it is intended to be invoked directly via invoke_action_addrs.
action "testing_action" "notify" {
  config {
    message = "directly invoked"
  }
}

# An ordinary resource that would otherwise be created during a normal plan.
# When the action is directly invoked, the plan runs in refresh-only mode for
# this component, so this resource change must be suppressed.
resource "testing_resource" "main" {
  value = "example"
}
