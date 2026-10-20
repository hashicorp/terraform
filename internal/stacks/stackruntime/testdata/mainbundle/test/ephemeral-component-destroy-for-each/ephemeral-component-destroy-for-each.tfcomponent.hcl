required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

locals {
  expansion_set = ["x", "y"]
}

provider "testing" "main" {}

component "a" {
  for_each = toset(local.expansion_set)

  source = "./a"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    name = each.key
  }
}

ephemeral_component "e" {
  for_each = toset(local.expansion_set)

  source = "./e"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    input = component.a[each.key].value
  }
}

provider "testing" "auth" {
  for_each = toset(local.expansion_set)

  config {
    authentication = ephemeral_component.e[each.key].value
    require_auth   = true
  }
}

component "b" {
  for_each = toset(local.expansion_set)

  source = "./b"

  providers = {
    testing = provider.testing.auth[each.key]
  }

  inputs = {
    name = each.key
  }
}
