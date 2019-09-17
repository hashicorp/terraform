provider "test" {
  alias = "baz"
}

resource "test_instance" "foo" {
  provider = "test.baz"
}

module "bar" {
  source = "./baz"

  providers = {
    test     = "test.baz"
    test.foo = "test"
  }
}
