variable "instance_count" {
  default = 1
}

resource "aws_instance" "test" {
  count = "${var.instance_count}"
}

resource "aws_instance" "dependent" {
  count = "${aws_instance.test.count}"
}
