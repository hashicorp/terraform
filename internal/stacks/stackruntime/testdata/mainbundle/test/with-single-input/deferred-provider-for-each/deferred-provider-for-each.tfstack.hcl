required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "providers" {
  type = set(string)
}

provider "testing" "unknown" {
  for_each = var.providers
}

provider "testing" "known" {
  for_each = toset(["primary"])
}

component "known" {
  source = "../"

  providers = {
    // We're testing a known component referencing an unknown provider here.
    testing = provider.testing.unknown["primary"]
  }

  inputs = {
    input = "primary"
  }
}

component "unknown" {
  source = "../"

  for_each = var.providers

  providers = {
    // We're testing an unknown component referencing a known provider here.
    testing = provider.testing.known[each.key]
  }

  inputs = {
    input = "secondary"
  }
}
