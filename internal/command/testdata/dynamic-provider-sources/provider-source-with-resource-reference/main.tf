resource "test_instance" "example" {}

terraform {
  required_providers {
    test = {
      source = "hashicorp/${test_instance.example.id}"
    }
  }
}
