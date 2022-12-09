variable "foo" {}

resource "test_instance" "foo" {
    foo = var.foo
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
