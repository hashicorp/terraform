resource "test_instance" "a" {
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
