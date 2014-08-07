resource "aws_instance" "foo" {
    num = "2"

    count = 1
}

resource "aws_instance" "bar" {
    foo = "${aws_instance.foo.*.num}"
}
