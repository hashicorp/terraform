provider "aws" {}

resource "aws_instance" "bar" {
    ami = "abc"
    lifecycle {
        create_before_destroy = true
    }
}
