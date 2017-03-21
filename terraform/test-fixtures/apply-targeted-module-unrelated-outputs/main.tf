resource "aws_instance" "foo" {}

module "child1" {
  source = "./child1"
  instance_id = "${aws_instance.foo.id}"
}

module "child2" {
  source = "./child2"
}
