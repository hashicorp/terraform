terraform {
  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "1.2.3"
    }
  }
  state_store "test_store" {
    provider "test" {}

    value = "foobar"
  }
}

provider "exact" {
  version = "1.2.3"
}

provider "greater-than" {
  version = ">= 2.3.3"
}

provider "between" {
  # The second constraint here intentionally has
  # no space after the < operator to make sure
  # that we can parse that form too.
  version = "> 1.0.0 , <3.0.0"
}
