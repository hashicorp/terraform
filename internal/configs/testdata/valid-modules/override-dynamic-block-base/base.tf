terraform {
  required_providers {
    test = {
      source = "terraform.io/test-only/test"
    }
  }
}

resource "test" "foo" {
  dynamic "foo" {
    for_each = []
    content {
      from = "base"
    }
  }
}
