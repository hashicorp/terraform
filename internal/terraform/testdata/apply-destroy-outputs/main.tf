data "test_data_source" "bar" {
  for_each = {
    a = "b"
  }
  foo = "zing"
}

data "test_data_source" "foo" {
  for_each = data.test_data_source.bar
  foo = "ok"
}

locals {
  l = [
    {
      name = data.test_data_source.foo["a"].id
      val = "null"
    },
  ]

  m = { for v in local.l :
    v.name => v
  }
}

resource "test_instance" "bar" {
  for_each = local.m
  foo = format("%s", each.value.name)
  dep = each.value.val
}

output "out" {
  value = test_instance.bar
}
