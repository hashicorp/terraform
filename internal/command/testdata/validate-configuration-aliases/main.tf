# This fixture represents a reusable (child) module which declares an
# additional provider configuration via configuration_aliases and uses it
# from a resource. Running "terraform validate" directly in this directory
# treats the module as the root module, so there is no parent module to pass
# the aliased provider configuration in, but validation must still succeed.
#
# See https://github.com/hashicorp/terraform/issues/28490
terraform {
  required_providers {
    test = {
      source                = "hashicorp/test"
      configuration_aliases = [test.alias]
    }
  }
}

resource "test_instance" "default" {
  ami = "bar"
}

resource "test_instance" "aliased" {
  provider = test.alias
  ami      = "baz"
}
