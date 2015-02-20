variable "count" {}

module "child" {
    source = "./child"
    count = "${var.count}"
}
