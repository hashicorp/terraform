import {
  for_each = ["a", "b"]
  to = test_resource.test[each.value]
  id = each.value
}

resource "test_resource" "test" {
  for_each = toset(["a", "b"])
}
