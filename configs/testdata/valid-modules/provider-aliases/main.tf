terraform {
  required_providers {
    foo-test = {
      source = "foo/test"
      // TODO: these are strings until the parsing code is refactored to allow
      // raw references
      configuration_aliases = [foo-test.a, foo-test.b]
    }
  }
}

resource "test_instance" "explicit" {
  provider = foo-test.a
}

data "test_resource" "explicit" {
  provider = foo-test.b
}

