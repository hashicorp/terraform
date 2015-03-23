provider "aws" {
	alias = "bar"
}

resource "aws_instance" "foo" {
    num = "2"
}

resource "aws_instance" "bar" {
    foo = "bar"
    provider = "aws.bar"
}
