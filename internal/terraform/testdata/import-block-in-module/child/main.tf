locals {
  ids = {
    first = "testa"
    second = "testb"
  }
}

resource test_object bar {
  for_each = local.ids
}

import {
  for_each = local.ids
  to = test_object.bar[each.key]
  id = each.value
}