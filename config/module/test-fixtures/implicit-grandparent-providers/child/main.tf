resource "foo_resource" "in_child" {}

provider "bar" {
    value = "from child"
}

module "grandchild" {
    source = "./grandchild"
}
