terraform {
  required_providers {
    foo = {
      source = "hashicorp/foo"
    }
    baz = {
      source = "hashicorp/baz"
    }
  }
}

module "mod" {
  source = "./mod"
  providers = {
    foo = foo
    baz = baz
  }
}
