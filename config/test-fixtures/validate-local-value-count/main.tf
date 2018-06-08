
locals {
  count = 3
}

resource "null_resource" "foo" {
  count = "${local.count}"
}
