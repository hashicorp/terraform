terraform {
  required_providers {
    foo = {
      source = "example.com/vendor/foo"
    }
  }
}

resource "foo_resource" "a" {
}

// implied default provider null
resource "null_resource" "a" {
}
