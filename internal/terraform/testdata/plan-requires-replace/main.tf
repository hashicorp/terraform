terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_thing" "foo" {
  v = "goodbye"
}
