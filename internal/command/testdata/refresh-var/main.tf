variable "foo" {}

provider "test" {
    value = "${var.foo}"
}

resource "test_instance" "foo" {}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
