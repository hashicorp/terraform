terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_instance" "test" {
  ami = "bar"
}

module "with_requirement" {
  source = "./nested"
}

module "no_requirements" {
  source = "./nested-no-requirements"
}
