resource "null_resource" "foo" {}

module "child1" {
  source = "./child1"
  instance_id = "${null_resource.foo.id}"
}

module "child2" {
  source = "./child2"
}
