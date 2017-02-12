variable "foo" {
    default = "2"
}

resource "aws_instance" "foo" {
    count = "${var.foo}"
}

resource "aws_instance" "bar" {
    foo = "${aws_instance.foo.count}"
}
