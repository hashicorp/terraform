variable "foo" {}

module "child" {
    source = "./child"
    foo = var.foo
}
