terraform {
  required_providers {
    test = {
      source = "terraform.io/builtin/test"
    }
  }
}

provider "test" {
  arg = "foo"
}

module "b" {
  source = "../module-b"
}
