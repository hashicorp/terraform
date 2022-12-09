terraform {
  required_providers {
    foo = {
      source = "terraform.io/test-only/foo"
    }
    bar = {
      source = "terraform.io/test-only/bar"
    }
  }
}

resource "foo_instance" "one" {}
resource "bar_instance" "two" {}
