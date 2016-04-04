variable "count" {
  default = 3
}

resource "aws_instance" "foo" {
  count = "${var.count}"
}
