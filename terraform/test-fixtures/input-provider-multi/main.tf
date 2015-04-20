provider "aws" {
    alias = "east"
}

resource "aws_instance" "foo" {
    alias = "east"
}

resource "aws_instance" "bar" {}
