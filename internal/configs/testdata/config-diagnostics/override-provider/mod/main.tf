terraform {
  required_providers {
    bar = {
      source  = "terraform.io/test-only/bar"
      version = "~>1.0.0"
    }
  }
}

// this configuration cannot be overridden from an outside module
provider "bar" {
  value = "ok"
}
