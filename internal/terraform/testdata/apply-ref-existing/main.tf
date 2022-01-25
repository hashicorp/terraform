resource "aws_instance" "foo" {
  foo = "bar"
}

module "child" {
  source = "./child"

  var = "${aws_instance.foo.foo}"
}
