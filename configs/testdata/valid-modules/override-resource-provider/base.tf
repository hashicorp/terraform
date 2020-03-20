terraform {
  required_providers {
    foo-test = {
      source = "-/foo"
    }
    bar-test = {
      source = "-/bar"
    }
  }
}

resource "test_instance" "explicit" {
  provider = foo-test
}

// the provider for this resource should default to "hashicorp/test"
resource "test_instance" "default" {}
