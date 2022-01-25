/*
module "child" {
  source = "./child"
}

resource "aws_instance" "bar" {
  childid      = "${module.child.id}"
  grandchildid = "${module.child.grandchild_id}"
}
*/
