resource "aws_instance" "foo" {
    count = 3
    value = "${aws_instance.foo.0.value}"
}
