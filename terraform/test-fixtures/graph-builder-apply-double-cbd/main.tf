resource "aws_instance" "A" {
  lifecycle { create_before_destroy = true }
}

resource "aws_instance" "B" {
  value = ["${aws_instance.A.*.id}"]

  lifecycle { create_before_destroy = true }
}
