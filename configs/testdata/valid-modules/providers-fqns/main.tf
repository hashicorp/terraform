terraform {
  required_providers {
    foo-test = {
      // This is depending on the current behavior which allows legacy-style
      // provider sources. When provider source is fully implemented this can be
      // update to use any namespace. 
      source = "-/test"
    }
  }
}

provider "foo-test" {}
