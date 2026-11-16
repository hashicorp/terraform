variable "root" {
  type = string
}

locals {
  root = "${var.root} -> local"
}

resource "test_resource" "root" {
  value = "resource attr set to local -> ${local.root}"
}

module "child_single" {
  source      = "./child"
  child_input = "${var.root} -> child"
}

module "child_foreach" {
  for_each    = toset(["key1", "key2"])
  source      = "./child"
  child_input = "${var.root} -> child[${each.key}]"
}

module "child_count" {
  count       = 2
  source      = "./child"
  child_input = "${var.root} -> child[${count.index}]"
}
