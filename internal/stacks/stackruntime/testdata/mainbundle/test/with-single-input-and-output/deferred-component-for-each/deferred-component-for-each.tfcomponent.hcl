required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "components" {
  type = set(string)
}

provider "testing" "default" {}

component "self" {
  // This component validates the behaviour of an unknown for_each value.

  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = each.value
  }

  for_each = var.components
}

component "child" {
  // This component validates the behaviour of referencing a partial component
  // with a known key. Since we don't know the available keys of the component
  // yet, this should use the outputs of the partial instance.

  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    // It's really unlikely that `const` is actually going to exist once
    // component.self has known keys, but for now we don't know it doesn't
    // exist so we should defer this component and make a reasonable attempt
    // at planning something.
    input = component.self["const"].id
  }
}
