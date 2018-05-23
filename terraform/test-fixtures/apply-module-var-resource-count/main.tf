variable "num" {
}

module "child" {
    source = "./child"
    num = "${var.num}"
}
