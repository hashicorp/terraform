variable "foo" {
  default = {}
}

locals {
  baz = var.foo.bar.baz
}

resource "test_instance" "foo" {
    foo = local.baz
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
