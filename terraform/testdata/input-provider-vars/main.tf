variable "foo" {}

resource "aws_instance" "foo" {
    foo = "${var.foo}"
}
