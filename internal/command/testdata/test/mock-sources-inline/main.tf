terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

provider "test" {
  alias = "secondary"
}

resource "test_resource" "foo" {
  value = "foo"
}

resource "test_resource" "bar" {
  provider = test.secondary
  value    = "bar"
}

output "foo" {
  value = test_resource.foo.id
}
output "bar" {
  value = test_resource.bar.id
}
