module "bar" {
    source = "./bar"
    bar = "${var.foo}"
}

variable "foo" {}
