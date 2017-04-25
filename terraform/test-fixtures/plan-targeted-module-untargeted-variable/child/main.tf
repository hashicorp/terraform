variable "id" {}

resource "aws_instance" "mod" {
  value = "${var.id}"
}
