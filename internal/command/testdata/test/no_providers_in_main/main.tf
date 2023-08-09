
terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      configuration_aliases = [test.primary, test.secondary]
    }
  }
}

resource "test_resource" "primary" {
  provider = test.primary
  value = "foo"
}

resource "test_resource" "secondary" {
  provider = test.secondary
  value = "bar"
}
