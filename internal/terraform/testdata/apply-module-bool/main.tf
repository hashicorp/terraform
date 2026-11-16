module "child" {
    source = "./child"
    leader = true
}

resource "aws_instance" "bar" {
    foo = "${module.child.leader}"
}
