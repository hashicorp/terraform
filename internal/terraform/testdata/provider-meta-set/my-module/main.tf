resource "test_resource" "bar" {
  value = "bar"
}

terraform {
  provider_meta "test" {
    baz = "quux-submodule"
  }
}
