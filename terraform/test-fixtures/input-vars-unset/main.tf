variable "foo" {}
variable "bar" {}

resource "aws_instance" "foo" {
    foo = "${var.foo}"
    bar = "${var.bar}"
}
