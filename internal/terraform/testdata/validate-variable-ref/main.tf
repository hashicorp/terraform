variable "foo" {}

resource "aws_instance" "bar" {
    foo = "${var.foo}"
}
