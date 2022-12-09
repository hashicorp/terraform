terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_instance" "bar" {
  foo = "bar"
}

module "my_module" {
  source = "./my-module"
}
