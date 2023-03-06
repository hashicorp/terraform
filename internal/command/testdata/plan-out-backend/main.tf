terraform {
  backend "http" {
  }
}

resource "test_instance" "foo" {
  ami = "bar"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
