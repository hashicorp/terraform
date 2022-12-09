terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

data "test_data_source" "foo" {
  foo = "bar"
}

module "my_module" {
  source = "./my-module"
}
