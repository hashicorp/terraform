variable "num" {
}

resource "aws_instance" "foo" {
  count = "${var.num}"
}
