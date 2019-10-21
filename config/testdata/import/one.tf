variable "bar" {}

provider "aws" {
    bar = "baz";
}

resource "aws_security_group" "db" {}
