resource "aws_instance" "foo" {}

module "child" {
    source = "./child"
    in = "${aws_instance.foo.id}"
}


