data "test_file" "foo" {
  id = "bar"
}

terraform {
  provider_meta "test" {
    baz = "quux-submodule"
  }
}
