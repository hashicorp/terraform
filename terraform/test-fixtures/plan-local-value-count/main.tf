
locals {
  count = 3
}

resource "test_resource" "foo" {
  count = "${local.count}"
}
