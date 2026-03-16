terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

provider "test" {}

resource "test_instance" "example" {
  ami = "ami-12345"
}
