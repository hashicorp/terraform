terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      version = "1.0.1"
    }
  }
}

resource "test_instance" "baz" {
  ami = "baz"
}
