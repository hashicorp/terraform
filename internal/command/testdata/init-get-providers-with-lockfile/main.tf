terraform {
  required_providers {
    exact = {
      source  = "hashicorp/exact"
      version = "1.2.3"
    }
    greater-than = {
      source  = "hashicorp/greater-than"
      version = ">= 2.3.3"
    }
    between = {
      source = "hashicorp/between"
      # The second constraint here intentionally has
      # no space after the < operator to make sure
      # that we can parse that form too.
      version = "> 1.0.0 , <3.0.0"
    }
  }
}
