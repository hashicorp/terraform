terraform {
  required_providers {
    test = {
      source = "test"
    }
  }
}

variable "" {

}

module "example" {
  source = provider::test::is_true(true) ? "./modules/example" : ""
}
