provider "test-beta" {
  foo = "baz"
}

resource "test_instance" "foo" {
  provider = "test-beta"
}
