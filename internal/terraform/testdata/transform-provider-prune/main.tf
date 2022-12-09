terraform {
  required_providers {
    foo = {
      source = "terraform.io/test-only/foo"
    }
  }
}

provider "aws" {}
resource "foo_instance" "web" {}
