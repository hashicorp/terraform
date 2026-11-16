resource "aws_instance" "foo" {
}

module "child" {
  source = "./child"
}

output "root_id" {
  value = "${aws_instance.foo.id}"
}

output "child_id" {
  value = "${module.child.id}"
}

output "grandchild_id" {
  value = "${module.child.grandchild_id}"
}
