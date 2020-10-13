provider "aws" {

}

provider "template" {
  alias = "foo"
}

resource "aws_instance" "foo" {

}

resource "null_resource" "foo" {

}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
