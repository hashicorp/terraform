variable "count" {}

resource "aws_instance" "foo" {
    count = "${var.count}"
    value = "foo"
}

resource "aws_instance" "bar" {
    value = "${join(",", aws_instance.foo.*.id)}"
}
