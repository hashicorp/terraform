terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

module "mod2" {
  source = "./mod2"

  // the test provider is named here, but a config must be supplied from the
  // parent module.
  providers = {
    test.foo = test
  }
}
