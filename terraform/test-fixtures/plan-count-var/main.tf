variable "count" {}

resource "aws_instance" "foo" {
    count = "${var.count}"
    foo = "foo"
}

resource "aws_instance" "bar" {
    foo = "${join(",", aws_instance.foo.*.foo)}"
}
