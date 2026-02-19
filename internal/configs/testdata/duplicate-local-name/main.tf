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

    wrong-name = {
      source = "hashicorp/foo"
    }
  }
}

provider "default" {
}

resource "foo_resource" {
}
