resource "aws_instance" "foo" {
    id = "foo"
    num = "2"
}

resource "aws_instance" "bar" {
    id = "bar"
    num = "2"
    foo = "${aws_instance.foo.id}"
}
