provider "foo" {
  version = "1.2.3"
}

terraform {
  required_providers {
    bar = {
      source  = "hashicorp/bar"
      version = "2.0.0"
    }
    baz = {
      source  = "terraform-providers/baz"
      version = "3.0.0"
    }
    foo = {
      source = "hashicorp/foo"
    }
  }
}

provider "terraform" {}
