terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_resource" "bar" {
  value = "bar"
}
