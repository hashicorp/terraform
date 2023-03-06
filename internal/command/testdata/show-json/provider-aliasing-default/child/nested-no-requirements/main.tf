resource "test_instance" "test" {
  ami = "qux"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
