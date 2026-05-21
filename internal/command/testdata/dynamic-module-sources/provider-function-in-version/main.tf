terraform {
  required_providers {
    test = {
      source = "test"
    }
  }
}

module "example" {
  source  = "./modules/example"
  version = provider::test::is_true(true) ? "1.0.0" : ""
}
