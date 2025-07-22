required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "providers" {
  type = set(string)
}

provider "testing" "main" {
  for_each = var.providers
}

provider "testing" "const" {}

component "main" {
  source = "../"

  for_each = var.providers

  providers = {
    // We're testing an unknown component referencing a known provider here.
    testing = provider.testing.main[each.key]
  }

  inputs = {
    id = "data_unknown"
    resource = "resource_unknown"
  }
}

component "const" {
  source = "../"

  providers = {
    testing = provider.testing.const
  }

  inputs = {
    id = "data_known"
    resource = "resource_known"
  }
}
