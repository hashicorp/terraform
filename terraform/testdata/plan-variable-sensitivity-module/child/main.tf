variable "foo" {
  type = string
}

resource "aws_instance" "foo" {
  foo = var.foo
}
