variable "foo" {}

provider "test" {
    foo = "${var.foo}"
}

resource "test_instance" "foo" {
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
