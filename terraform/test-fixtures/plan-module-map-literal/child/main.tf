variable "amap" {
  type = "map"
}

variable "othermap" {
  type = "map"
}

resource "aws_instance" "foo" {
  tags = "${var.amap}"
  meta = "${var.othermap}"
}
