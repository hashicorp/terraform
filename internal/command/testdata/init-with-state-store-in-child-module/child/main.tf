terraform {
  required_providers {
    test = {
      source  = "hashicorp/test"
      version = ">=2.0.0"
    }
  }
}

resource "test_resource" "example" {
  string = "Hello, world!"
}
