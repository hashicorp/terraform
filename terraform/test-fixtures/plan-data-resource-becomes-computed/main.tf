resource "aws_instance" "foo" {
}

data "aws_data_resource" "foo" {
  foo = "${aws_instance.foo.computed}"
}
