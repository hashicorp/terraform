terraform {
  required_providers {
    test = {
      source = "terraform.io/test-only/test"
    }
  }
}

resource "test" "foo" {
  foo {
    from = "base"
  }
}
