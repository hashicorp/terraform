resource "aws_instance" "A" {
  count = 1
}

resource "aws_instance" "B" {
  value = ["${aws_instance.A.*.id}"]
}
