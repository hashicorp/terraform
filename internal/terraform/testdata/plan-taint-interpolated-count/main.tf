variable "instance_count" {
  default = 3
}

resource "aws_instance" "foo" {
  count = "${var.instance_count}"
}
