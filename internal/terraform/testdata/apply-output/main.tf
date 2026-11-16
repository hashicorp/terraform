resource "aws_instance" "foo" {
    num = "2"
}

resource "aws_instance" "bar" {
    foo = "bar"
}

output "foo_num" {
    value = "${aws_instance.foo.num}"
}
