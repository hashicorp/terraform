resource "test_resource" "bar" {
  value = "bar"
}

mnptu {
  provider_meta "test" {
    baz = "quux-submodule"
  }
}
