resource "aws_instance" "foo" {
    num = "2"
    compute = "id"
    compute_value = "${var.value}"
}

resource "aws_instance" "bar" {
    foo = "${aws_instance.foo.id}"
}
