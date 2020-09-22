provider "foo" {
  version = "1.2.3"
}

terraform {
  required_providers {
    bar = "1.0.0"
    baz = {
      version = "~> 2.0.0"
    }
  }
}

provider "terraform" { }
