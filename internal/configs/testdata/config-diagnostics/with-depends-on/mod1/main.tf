terraform {
  required_providers {
    foo = {
      source = "hashicorp/foo"
    }
  }
}

resource "foo_resource" "a" {
}

module "mod2" {
  depends_on = [foo_resource.a]
  // test fixture source is from root
  source = "./mod1/mod2"
  providers = {
    foo = foo
  }
}
