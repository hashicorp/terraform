resource "test_instance" "bar" {
  foo = "bar"
}

terraform {
  provider_meta "test" {
    baz = "quux"
  }
}

module "my_module" {
  source = "./my-module"
}
