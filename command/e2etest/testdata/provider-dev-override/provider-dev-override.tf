terraform {
  required_providers {
    test = {
      source  = "example.com/test/test"
      version = "2.0.0"
    }
  }
}

provider "test" {
}

data "test_data_source" "test" {
}
