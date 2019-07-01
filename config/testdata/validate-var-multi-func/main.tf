resource "aws_instance" "foo" {
    count = 3
}

resource "aws_instance" "bar" {
    foo = "${element(aws_instance.foo.*.id, 0)}"
}
