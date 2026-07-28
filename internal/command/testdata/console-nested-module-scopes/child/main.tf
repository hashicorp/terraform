variable "child_input" {
  type = string
}

locals {
  child = "${var.child_input} -> local"
}

resource "test_resource" "child" {
  value = "resource attr set to local -> ${local.child}"
}

module "grandchild" {
  for_each         = toset(["key"])
  source           = "./grandchild"
  grandchild_input = "${var.child_input} -> grandchild[${each.key}]"
}

output "child_output" {
  value = "child module output"
}
