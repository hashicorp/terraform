terraform {
  required_providers {
    bar-test = {
      source = "bar/test"
    }
    foo-test = {
      source = "foo/test"
      configuration_aliases = [foo-test.other]
    }
  }
}

resource "test_instance" "explicit" {
  // explicitly setting provider bar-test
  provider = bar-test
}

resource "test_instance" "implicit" {
  // since the provider type name "test" does not match an entry in
  // required_providers, the default provider "test" should be used
}

resource "test_instance" "other" {
  provider = foo-test.other
}
