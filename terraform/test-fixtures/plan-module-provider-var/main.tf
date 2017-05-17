variable "foo" { default = "bar" }

module "child" {
    source = "./child"
    foo    = "${var.foo}"
}
