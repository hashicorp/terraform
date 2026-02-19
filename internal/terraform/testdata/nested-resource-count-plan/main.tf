resource "aws_instance" "foo" {
  count = 2
}

resource "aws_instance" "bar" {
  count = "${length(aws_instance.foo.*.id)}"
}

resource "aws_instance" "baz" {
  count = "${length(aws_instance.bar.*.id)}"
}
