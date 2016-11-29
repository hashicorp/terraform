variable "ami" {
  default = "foo"
  type    = "string"
}

variable "list" {
  default = []
  type    = "list"
}

variable "map" {
  default = {}
  type = "map"
}

resource "aws_instance" "bar" {
  foo = "${var.ami}"
  bar = "${join(",", var.list)}"
  baz = "${join(",", keys(var.map))}"
}
