terraform {
  required_providers {
    foo-test = {
      source = "foo/test"
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

