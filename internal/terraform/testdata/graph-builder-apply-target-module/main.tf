resource "test_object" "foo" {}

module "child1" {
  source      = "./child1"
  instance_id = "${test_object.foo.id}"
}

module "child2" {
  source = "./child2"
}
