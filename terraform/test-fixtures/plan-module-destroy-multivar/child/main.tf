variable "instance_count" {
  default = "1"
}

resource "aws_instance" "foo" {
  count = "${var.instance_count}"
  bar = "bar"
}
