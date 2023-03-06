variable "foo" {}

resource "test_instance" "foo" {
    value = "${var.foo}"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
