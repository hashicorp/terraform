variable "foo" {
    default = "bar"
}

provider "aws" {
    set = "${var.foo}"
}

resource "aws_instance" "foo" {}
