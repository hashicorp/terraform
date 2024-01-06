terraform {
  required_providers {
    test = {
      source = "terraform.io/builtin/test"
    }
  }
}

provider "test" {
  arg = "foo"
}

module "b" {
  # FIXME: The following is an absolute remote address only because at the
  # time of writing this test the stacks runtime's module loader can't deal
  # with relative paths in this location.
  source = "https://testing.invalid/validating.tar.gz//modules_with_provider_configs/module-b"
  # Once that's been fixed, this should instead be:
  # source = "../module-b"
}
