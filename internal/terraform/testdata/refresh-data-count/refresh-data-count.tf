terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test" "foo" {
}

data "test" "foo" {
  count = length(test.foo.things)
}
