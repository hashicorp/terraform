terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

provider "test" {}
