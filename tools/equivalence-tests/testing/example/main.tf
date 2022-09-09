terraform {
  required_providers {
    mock = {
      source = "liamcervante/mock"
    }
  }
}

provider "mock" {}

resource "mock_simple_resource" "integer" {
  id = "my_integer"
  integer = 1
}
