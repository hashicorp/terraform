variable "value" {}

provider "aws" {
    value = "${var.value}"
}

resource "aws_instance" "foo" {}
