terraform {
  required_providers {
    test = {
      source = "terraform.io/test-only/test"
    }
  }
}

resource "test_instance" "foo" {
  foo = "bar"
}
