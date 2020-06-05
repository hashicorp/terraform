provider "aws" {
  alias = "mod"
}

resource "aws_instance" "bar" {
  provider = "aws.mod"
}
