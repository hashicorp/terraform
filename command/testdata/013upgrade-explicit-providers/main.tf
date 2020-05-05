provider foo {}

terraform {
  required_providers {
    bar = "1.0.0"
    baz = {
      version = "~> 2.0.0"
    }
  }
}
