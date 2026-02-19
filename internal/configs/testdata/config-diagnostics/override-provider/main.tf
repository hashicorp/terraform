terraform {
  required_providers {
    bar = {
      version = "~>1.0.0"
    }
  }
}

provider "bar" {
  value = "not ok"
}

// this module configures its own provider, which cannot be overridden
module "mod" {
  source = "./mod"
  providers = {
    bar = bar
  }
}
