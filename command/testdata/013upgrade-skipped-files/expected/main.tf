provider foo {
}

terraform {
  required_providers {
    bar = {
      source  = "hashicorp/bar"
      version = "1.0.0"
    }
    baz = {
      source  = "terraform-providers/baz"
      version = "~> 2.0.0"
    }
    foo = {
      source  = "hashicorp/foo"
      version = "1.2.3"
    }
  }
}
