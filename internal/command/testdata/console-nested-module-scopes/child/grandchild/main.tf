variable "grandchild_input" {
  type = string
}

locals {
  grandchild = "${var.grandchild_input} -> local"
}

resource "test_resource" "grandchild" {
  value = "resource attr set to local -> ${local.grandchild}"
}

output "grandchild_output" {
  value = "grandchild module output"
}
