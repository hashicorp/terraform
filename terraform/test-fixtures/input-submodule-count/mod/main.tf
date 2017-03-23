variable "count" {
}

resource "aws_instance" "foo" {
  count = "${var.count}"
}

module "submod" {
  source = "./submod"
  list = ["${aws_instance.foo.*.id}"]
}
