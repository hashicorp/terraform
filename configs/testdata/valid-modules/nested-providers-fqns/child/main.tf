terraform {
  required_providers {
    bar-test = {
      source = "bar/test"
    }
  }
}

provider "bar-test" {}
