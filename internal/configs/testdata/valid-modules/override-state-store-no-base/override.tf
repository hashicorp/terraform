terraform {
  // Note: not valid config - a paired entry in required_providers is usually needed
  state_store "foo_bar" {
    provider "foo" {}

    custom_attr = "foobar"
  }
}

provider "foo" {}
