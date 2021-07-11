resource "null_resource" "foo" {}

resource "null_resource" "bar" {}

module "child" {
  source = "./child"
}
