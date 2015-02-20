resource "aws_instance" "foo" {
    count = -5
    value = "${aws_instance.foo.0.value}"
}
