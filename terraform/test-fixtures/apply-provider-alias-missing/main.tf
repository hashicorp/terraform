provider "aws" {
	alias = "bar"
}

resource "aws_instance" "bar" {
    foo = "bar"
    provider = "aws.bar"
}
