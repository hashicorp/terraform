terraform {

  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "<2.0.0" // mutually exclusive with the version constraint in the child module, which should cause an error during init
    }
  }
  state_store "test_store" {
    provider "test" {
    }

    value = "foobar"
  }
}

module "child" {
  source = "./child"
}
