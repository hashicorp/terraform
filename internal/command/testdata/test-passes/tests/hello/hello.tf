terraform {
  required_providers {
    test = {
      source = "terraform.io/builtin/test"
    }
  }
}

module "main" {
  source = "../.."

  input = "boop"
}

resource "test_assertions" "foo" {
  component = "foo"

  equal "output" {
    description = "output \"foo\" value"
    got         = module.main.foo
    want        = "foo value boop"
  }
}
