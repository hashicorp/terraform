variable "count" {}

resource "aws_instance" "foo" {
  count = "${var.count}"
}
