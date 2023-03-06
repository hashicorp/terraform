terraform {
  required_providers {
    foo = {
      source = "terraform.io/test-only/foo"
    }
  }
}

provider "foo" {}

module "subsub" {
    source = "./subsub"
}
