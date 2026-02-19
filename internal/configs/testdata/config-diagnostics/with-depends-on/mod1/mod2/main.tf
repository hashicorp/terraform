terraform {
  required_providers {
    foo = {
      source = "hashicorp/foo"
    }
  }
}

module "mod3" {
  // test fixture source is from root
  source = "./mod1/mod2/mod3"
  providers = {
    foo.bar = foo
  }
}
