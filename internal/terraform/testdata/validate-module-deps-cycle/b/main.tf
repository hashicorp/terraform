variable "input" {}

resource "aws_instance" "b" {
  id = "${var.input}"
}
