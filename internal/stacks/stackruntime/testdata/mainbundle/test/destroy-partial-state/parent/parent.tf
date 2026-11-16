terraform {
  required_providers {
    testing = {
      source = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

resource "testing_resource" "deleted" {}

resource "testing_resource" "primary" {}

resource "testing_resource" "secondary" {
  value = testing_resource.primary.id
}

resource "testing_resource" "depends_on_deleted" {
  for_each = {
    (testing_resource.deleted.id) = "tertiary"
  }
  id = each.value
}

output "deleted_id" {
  value = testing_resource.deleted.id
}
