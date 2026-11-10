terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

# The ids are unknown at plan.
resource "testing_resource" "good" {
  value = "ok"
}

resource "testing_resource" "bad" {
  value = "bad"
}

output "good_id" {
  value = testing_resource.good.id
}

output "bad_id" {
  value = testing_resource.bad.id
}
