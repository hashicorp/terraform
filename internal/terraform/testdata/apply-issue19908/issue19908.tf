terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test" "foo" {
  baz = "updated"
}
