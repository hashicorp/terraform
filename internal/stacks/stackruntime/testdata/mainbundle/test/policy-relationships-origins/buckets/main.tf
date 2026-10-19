terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

resource "testing_resource" "good" {
  id    = "shared"
  value = "ok"
}

resource "testing_resource" "bad" {
  id    = "shared"
  value = "bad"
}

output "good_id" {
  value = testing_resource.good.id
}

output "bad_id" {
  value = testing_resource.bad.id
}
