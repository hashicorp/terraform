variable "foo" { default = "bar" }

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {
      region = var.foo
    }

    value = "hardcoded"
  }
}
