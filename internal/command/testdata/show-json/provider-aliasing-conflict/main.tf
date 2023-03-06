provider "test" {
  region = "somewhere"
}

resource "test_instance" "test" {
  ami = "foo"
}

module "child" {
  source = "./child"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
