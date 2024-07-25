# This is an intentionally-minimal module just to give us something to
# point a component block at.

terraform {
  required_providers {
    happycloud = {
      source = "example.com/test/happycloud"
    }
  }
}

resource "happycloud_thingy" "foo" {
}
