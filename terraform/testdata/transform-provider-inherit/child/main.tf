provider "aws" {
    alias = "bar"
}

resource "aws_instance" "thing" {
    provider = aws.bar
}
