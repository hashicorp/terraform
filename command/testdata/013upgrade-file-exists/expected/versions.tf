provider foo {}
provider bar {}
terraform {
  required_providers {
    bar = {
      source = "hashicorp/bar"
    }
    foo = {
      source = "hashicorp/foo"
    }
  }
  required_version = ">= 0.13"
}
