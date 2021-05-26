terraform {
  required_providers {
    simple = {
      source  = "example.com/test/test"
      version = "2.0.0"
    }
  }
}

data "simple_resource" "test" {
}
