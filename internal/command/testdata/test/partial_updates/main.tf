
resource "test_resource" "resource" {}

locals {
  follow = {
    (test_resource.resource.id): "follow"
  }
}

resource "test_resource" "follow" {
  for_each = local.follow

  id = each.key
  value = each.value
}
