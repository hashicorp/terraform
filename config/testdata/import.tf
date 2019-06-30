variable "foo" {
    default = "bar"
    description = "bar"
}

provider "aws" {
    foo = "bar"
}

resource "aws_security_group" "web" {}
