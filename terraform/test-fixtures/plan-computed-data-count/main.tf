resource "aws_instance" "foo" {
  num     = "2"
  compute = "foo"
}

data "aws_vpc" "bar" {
  count = 3
  foo = "${aws_instance.foo.foo}"
}
