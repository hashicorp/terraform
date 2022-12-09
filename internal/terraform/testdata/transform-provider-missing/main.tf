terraform {
  required_providers {
    foo = {
      source = "terraform.io/test-only/foo"
    }
  }
}

provider "aws" {}
resource "aws_instance" "web" {}
resource "foo_instance" "web" {}
