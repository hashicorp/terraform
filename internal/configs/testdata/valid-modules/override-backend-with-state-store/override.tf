terraform {
  # Not including an experiments list here
  # See https://github.com/hashicorp/terraform/issues/38012
  required_providers {
    foo = {
      source = "my-org/foo"
    }
  }
  state_store "foo_override" {
    provider "foo" {}

    custom_attr = "override"
  }
}
