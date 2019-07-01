variable "foo" {}

module "child" {
    source = "./child"

    value = "${var.foo}"
}

resource "aws_instance" "foo" {}
