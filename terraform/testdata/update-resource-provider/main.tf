provider "aws" {
  alias = "foo"
}

resource "aws_instance" "bar" {
  provider = "aws.foo"
}
