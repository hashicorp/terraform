terraform {
  required_providers {
    bar = {
      source = "hashicorp/bar"
    }
    baz = {
      source = "terraform-providers/baz"
    }
    foo = {
      source = "hashicorp/foo"
    }
  }
  required_version = ">= 0.13"
}
