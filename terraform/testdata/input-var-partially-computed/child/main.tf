variable "in" {}

resource "aws_instance" "mod" {
  value = "${var.in}"
}
