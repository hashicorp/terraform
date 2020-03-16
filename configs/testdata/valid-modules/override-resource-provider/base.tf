terraform {
  required_providers {
    foo-test = {
      source = "foo/test"
    }
    bar-test = {
      source = "bar/test"
    }
  }
}

resource "test_instance" "explicit" {
  provider = foo-test
}
