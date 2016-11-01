variable "value" {}

module "child" {
    source = "./child"
    value  = "${var.value}"
}
