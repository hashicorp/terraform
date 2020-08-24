terraform {
  required_providers {
    baz = {
      version = "~> 2.0.0"
    }
    foo = "< 2.0.0"
  }
}
