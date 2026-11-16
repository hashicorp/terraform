terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      version = "1.0.0"
    }
  }
}

resource "test_instance" "bar" {
  ami = "foo"
}
