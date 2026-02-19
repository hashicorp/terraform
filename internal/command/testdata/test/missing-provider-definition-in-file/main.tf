terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      configuration_aliases = [ test.secondary, test ]
    }
  }
}

resource "test_resource" "primary" {
  value = "foo"
}

resource "test_resource" "secondary" {
  provider = test.secondary
  value = "bar"
}
