terraform {
  required_providers {
    bar = {
      source = "registry.acme.corp/acme/bar"
    }
    baz = {
      source  = "terraform-providers/baz"
      version = "~> 2.0.0"
    }
    foo = {
      source  = "hashicorp/foo"
      version = "0.5"
    }
  }
  required_version = ">= 0.13"
}
