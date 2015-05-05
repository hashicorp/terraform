variable "input" {}

resource "aws_instance" "b" {
  name = "${var.input}"
}
