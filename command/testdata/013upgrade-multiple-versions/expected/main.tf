terraform {
  required_providers {
    baz = {
      source  = "terraform-providers/baz"
      version = "~> 2.0.0"
    }
    foo = {
      source  = "hashicorp/foo"
      version = "< 2.0.0,~> 1.2.3"
    }
  }
}
