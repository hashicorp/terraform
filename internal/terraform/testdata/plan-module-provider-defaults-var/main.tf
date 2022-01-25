module "child" {
    source = "./child"
}

provider "aws" {
    from = "${var.foo}"
}

resource "aws_instance" "foo" {}

variable "foo" {}
