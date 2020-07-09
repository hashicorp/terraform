data "test_data_source" "foo" {
  foo = "ok"
}

locals {
  l = [
    {
      name = data.test_data_source.foo.id
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
