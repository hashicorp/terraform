terraform {
  # Not including an experiments list here
  # See https://github.com/hashicorp/terraform/issues/38012
  required_providers {
    bar = {
      source = "my-org/bar"
    }
  }
  state_store "foo_override" {
    provider "bar" {}

    custom_attr = "override"
  }
}

provider "bar" {}
