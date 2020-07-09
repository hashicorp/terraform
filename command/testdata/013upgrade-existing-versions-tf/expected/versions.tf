# This is a file called versions.tf which does not originally have a
# required_providers block. 
resource foo_resource a {}

terraform {
  required_version = ">= 0.13"
  required_providers {
    bar = {
      source = "hashicorp/bar"
    }
    baz = {
      source = "terraform-providers/baz"
    }
    foo = {
      source = "hashicorp/foo"
    }
  }
}
