required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}


provider "testing" "default" {}


component "parent" {
  source = "./parent"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
      input = "parent"
  }
}


component "self" {
  source = "./self"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = each.value
  }

  for_each = component.parent.letters_in_id // This is a list and no set or map
}
