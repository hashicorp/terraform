variable "vpc_id" {}

resource "aws_instance" "child" {
  vpc_id = "${var.vpc_id}"
}
