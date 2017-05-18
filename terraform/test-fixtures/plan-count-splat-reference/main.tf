resource "aws_instance" "foo" {
    name = "foo ${count.index}"
    count = 3
}

resource "aws_instance" "bar" {
    foo_name = "${aws_instance.foo.*.name[count.index]}"
    count = 3
}
