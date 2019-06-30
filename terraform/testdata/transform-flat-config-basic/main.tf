resource "aws_instance" "foo" {}

resource "aws_instance" "bar" {
  value = "${aws_instance.foo.value}"
}

module "child" {
  source = "./child"
}
