terraform {
  // Note: not valid config - a paired entry in required_providers is usually needed
  state_store "foo_override" {
    provider "foo" {}

    custom_attr = "override"
  }
}

provider "bar" {}
