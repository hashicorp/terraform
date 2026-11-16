terraform {
  required_providers {
    foo = {
      source = "hashicorp/foo"
      configuration_aliases = [ foo.bar ]
    }
  }
}

resource "foo_resource" "a" {
  providers = foo.bar
}
