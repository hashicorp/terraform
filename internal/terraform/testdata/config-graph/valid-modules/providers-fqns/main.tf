terraform {
  required_providers {
    foo-test = {
      source = "foo/test"
    }
  }
}

provider "foo-test" {}
