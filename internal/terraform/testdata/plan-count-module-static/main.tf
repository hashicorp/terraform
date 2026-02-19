variable "foo" {
  default = "3"
}

module "child" {
  source = "./child"
  value  = "${var.foo}"
}
