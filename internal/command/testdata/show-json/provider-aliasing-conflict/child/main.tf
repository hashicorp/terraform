terraform {
  required_providers {
    test = {
      source = "hashicorp2/test"
    }
  }
}

resource "test_instance" "test" {
  ami = "bar"
}
