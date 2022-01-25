resource "aws_instance" "foo" {
}

module "grandchild" {
  source = "./grandchild"
}

output "id" {
  value = "${aws_instance.foo.id}"
}

output "grandchild_id" {
  value = "${module.grandchild.id}"
}
