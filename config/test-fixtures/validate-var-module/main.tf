module "foo" {}

resource "aws_instance" "foo" {
    foo = "${module.foo.bar}"
}
