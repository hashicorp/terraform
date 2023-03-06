terraform {
  required_providers {
    foo-test = {
      source = "foo/test"
    }
    bar-test = {
      source = "bar/test"
    }
  }
}

resource "aws_instance" "explicit" {
  provider = foo-test
}

// the provider for this resource should default to "hashicorp/aws"
resource "aws_instance" "default" {}
