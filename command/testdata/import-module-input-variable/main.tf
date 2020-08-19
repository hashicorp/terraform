variable "foo" {
  default = {}
}

module "child" {
    source = "./child"
    foo = var.foo
}
