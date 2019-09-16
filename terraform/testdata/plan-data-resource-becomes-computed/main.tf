resource "aws_instance" "foo" {
}

data "aws_data_source" "foo" {
  foo = "${aws_instance.foo.computed}"
}
