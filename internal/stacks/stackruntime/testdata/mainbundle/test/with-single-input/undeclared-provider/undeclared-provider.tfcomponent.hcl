variable "input" {
  type = string
}

component "self" {
  source = "../"

  providers = {
    # We haven't provided a definition for this anywhere.
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
  }
}

removed {
  from = component.removed

  source = "../"

  providers = {
    # We haven't provided a definition for this anywhere.
    testing = provider.testing.default
  }
}
