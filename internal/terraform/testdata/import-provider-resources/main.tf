terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

provider "aws" {
    value = "${test_instance.bar.id}"
}

resource "aws_instance" "foo" {
    bar = "value"
}

resource "test_instance" "bar" {
    value = "yes"
}
