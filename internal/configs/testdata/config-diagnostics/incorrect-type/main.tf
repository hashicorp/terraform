terraform {
  required_providers {
    foo = {
      source = "hashicorp/foo"
    }
    null = {
      source = "hashicorp/null"
    }
  }
}

module "mod" {
  source = "./mod"
  providers = {
    foo = foo
    null = null
  }
}
