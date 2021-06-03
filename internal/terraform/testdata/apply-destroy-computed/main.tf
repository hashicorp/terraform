resource "aws_instance" "foo" {}

module "child" {
    source = "./child"
    value = "${aws_instance.foo.output}"
}
