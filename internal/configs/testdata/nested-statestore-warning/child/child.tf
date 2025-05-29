terraform {
  # Only the root module can declare a state store. Terraform should emit a warning
  # about this child module state store declaration.
  required_providers {
    foobar = {
      source = "registry.terraform.io/my-org/foobar"
    }
  }
  state_store "ignored" {
    provider = foobar
  }
}
