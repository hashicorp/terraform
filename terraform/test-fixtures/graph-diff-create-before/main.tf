provider "aws" {}

resource "aws_instance" "bar" {
    ami = "abc"
    create_before_destroy = true
}
