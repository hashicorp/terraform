terraform {
  required_providers {
    test = {
      source                = "hashicorp/test"
      configuration_aliases = [test, test.second]
    }
  }
}

resource "test_instance" "test_primary" {
  ami      = "primary"
  provider = test
}

resource "test_instance" "test_secondary" {
  ami      = "secondary"
  provider = test.second
}

module "grandchild" {
  source = "./nested"
  providers = {
    test     = test
    test.alt = test.second
  }
}
