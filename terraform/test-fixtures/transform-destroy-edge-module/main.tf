resource "aws_instance" "a" {
    value = "${module.child.output}"
}

module "child" {
    source = "./child"
}
