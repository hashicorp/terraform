provider "aws" {
  foo = "bar"
  alias = "alias"
}

resource "aws_instance" "foo" {
  provider = "aws.alias"
}
