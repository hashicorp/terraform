resource "test_instance" "foo" {
  num = 2
}

resource "test_instance" "bar" {
  foo = "bar"
}

terraform {
  provider_meta "test" {
    baz = "quux-submodule"
  }
}
