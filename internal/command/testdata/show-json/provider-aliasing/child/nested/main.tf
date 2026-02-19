terraform {
  required_providers {
    test = {
      source                = "hashicorp/test"
      configuration_aliases = [test, test.alt]
    }
  }
}

resource "test_instance" "test_main" {
  ami      = "main"
  provider = test
}

resource "test_instance" "test_alternate" {
  ami      = "secondary"
  provider = test.alt
}
