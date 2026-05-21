terraform {
  required_providers {
    test = {
      source = "test"
    }
  }
}

module "example" {
  source = provider::test::is_true(true) ? "./modules/example" : ""
}
