variable "amap" {
  type = map(string)
}

variable "othermap" {
  type = map(string)
}

resource "aws_instance" "foo" {
  tags = "${var.amap}"
  meta = "${var.othermap}"
}
