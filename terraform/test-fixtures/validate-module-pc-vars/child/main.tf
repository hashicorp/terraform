variable "value" {}

provider "aws" {
    foo = "${var.value}"
}

resource "aws_instance" "foo" {}
