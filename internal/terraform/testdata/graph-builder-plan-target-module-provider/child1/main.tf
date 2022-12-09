terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

variable "key" {}

provider "test" {
  test_string = "${var.key}"
}

resource "test_object" "foo" {}
