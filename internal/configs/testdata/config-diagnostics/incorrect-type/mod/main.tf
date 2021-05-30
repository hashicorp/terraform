terraform {
  required_providers {
    foo = {
      source = "example.com/vendor/foo"
    }
  }
}

resource "foo_resource" "a" {
}

// implied default provider baz
resource "baz_resource" "a" {
}
