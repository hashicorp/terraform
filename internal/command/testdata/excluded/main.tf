resource "test_instance" "foo" {}

resource "test_instance" "bar" {
  depends_on = [test_instance.foo]
}

resource "test_instance" "baz" {
  depends_on = [test_instance.bar]
}

resource "test_instance" "quux" {
  depends_on = [test_instance.baz]
}
