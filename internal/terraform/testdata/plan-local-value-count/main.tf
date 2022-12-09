terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

locals {
  count = 3
}

resource "test_resource" "foo" {
  count = "${local.count}"
}
