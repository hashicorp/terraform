module "child" {
    source = "./child"
}

resource "aws_instance" "foo" {
    value = "${module.child.bar}"
}
