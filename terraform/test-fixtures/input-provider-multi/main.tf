provider "aws" {
  alias = "east"
}

resource "aws_instance" "foo" {
  provider = aws.east
}

resource "aws_instance" "bar" {}
