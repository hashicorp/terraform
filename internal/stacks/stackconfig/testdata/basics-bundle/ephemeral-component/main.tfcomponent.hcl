required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

variable "ids" {
  type = set(string)
}

ephemeral_component "basic" {
  source = "./ephemeral-output"

  providers = {
    testing = provider.testing.main
  }
}

ephemeral_component "expanded" {
  source   = "./ephemeral-output"
  for_each = var.ids

  inputs = {
    id = each.value
  }

  providers = {
    testing = provider.testing.main
  }

  depends_on = [ephemeral_component.basic]
}

output "value" {
  type      = string
  value     = ephemeral_component.basic.value
  ephemeral = true
}
