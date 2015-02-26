resource "aws_instance" "foo" {
    id = "foo"
    num = "2"
}

resource "aws_instance" "bar" {
    id = "bar"
    foo = "${aws_instance.foo.id}"
    require_new = "yes"
}
