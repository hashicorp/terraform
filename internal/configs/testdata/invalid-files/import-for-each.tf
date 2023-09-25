import {
  for_each = ["a", "b"]
  to = invalid[each.value]
  id = each.value
}

resource "test_resource" "test" {
  for_each = toset(["a", "b"])
}
