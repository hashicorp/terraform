terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

variable "should_ask" {}

provider "test" {
  value = var.should_ask
}

resource "test_instance" "foo" {
  foo = "bar"
}
