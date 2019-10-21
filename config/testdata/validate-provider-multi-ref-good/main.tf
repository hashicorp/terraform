provider "aws" {
    alias = "bar"
}

resource "aws_instance" "foo" {
    provider = "aws.bar"
}
