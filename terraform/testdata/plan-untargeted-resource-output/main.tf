module "mod" {
  source = "./mod"
}


resource "aws_instance" "c" {
  name = "${module.mod.output}"
}
