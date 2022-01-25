terraform {
  required_providers {
    foo-test = {
      source = "foo/test"
    }
  }
}

provider "foo-test" {}

module "child" {
  source = "./child"
  providers = {
    foo-test.other = foo-test
  }
}

resource "test_instance" "explicit" {
  provider = foo-test
}

data "test_resource" "explicit" {
  provider = foo-test
}

resource "test_instance" "implicit" {
  // since the provider type name "test" does not match an entry in
  // required_providers, the default provider "test" should be used
}
