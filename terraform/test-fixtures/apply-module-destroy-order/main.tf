module "child" {
  source = "./child"
}

resource "aws_instance" "b" {
    blah = "${module.child.a_output}"
}
