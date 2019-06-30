variable "foo" {
  default = 123
}

resource "aws_instance" "foo" {
  foo = "${var.foo}"
}
