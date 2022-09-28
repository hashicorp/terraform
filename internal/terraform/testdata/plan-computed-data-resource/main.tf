resource "aws_instance" "foo" {
  num     = "2"
  compute = "foo"
}

data "aws_vpc" "bar" {
  foo = "${aws_instance.foo.foo}"
}
