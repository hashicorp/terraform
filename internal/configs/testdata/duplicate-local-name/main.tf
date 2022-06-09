terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
    dupe = {
      source = "hashicorp/test"
    }
    other = {
      source = "hashicorp/default"
    }
  }
}

provider "default" {
}
