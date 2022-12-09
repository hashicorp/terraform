terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_thing" "zero" {
  count = 0
}

resource "test_thing" "one" {
  count = 1
}
