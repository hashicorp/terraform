resource "test_resource" "a" {
}

data "test_data" "d" {
  count = 1
  depends_on = [
    test_resource.a
  ]
}

resource "test_resource" "b" {
  count = 1
  foo = data.test_data.d[count.index].compute
}
