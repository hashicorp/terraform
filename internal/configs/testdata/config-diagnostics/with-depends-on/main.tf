terraform {
  required_providers {
    foo = {
      source = "hashicorp/foo"
    }
  }
}

module "mod2" {
  source = "./mod1"
  providers = {
    foo = foo
  }
}
