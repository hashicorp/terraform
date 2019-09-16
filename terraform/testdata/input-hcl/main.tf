variable "mapped" {
    type = "map"
}

variable "listed" {
    type = "list"
}

resource "hcl_instance" "hcltest" {
    foo = "${var.listed}"
    bar = "${var.mapped}"
}
