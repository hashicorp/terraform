terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "aws_instance" "foo" {}
resource "test_instance" "foo" {}
