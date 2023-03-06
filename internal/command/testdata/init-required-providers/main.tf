terraform {
  required_providers {
    null = "1.2.3"
    source = {
      source  = "hashicorp/source"
      version = "1.2.3"
    }
    test-beta = {
      source  = "hashicorp/test-beta"
      version = "1.2.4"
    }
  }
}
