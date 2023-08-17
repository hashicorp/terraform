data "test_file" "foo" {
  id = "bar"
}

mnptu {
  provider_meta "test" {
    baz = "quux-submodule"
  }
}
