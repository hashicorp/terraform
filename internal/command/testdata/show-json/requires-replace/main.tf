resource "test_instance" "test" {
    ami = "force-replace"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
