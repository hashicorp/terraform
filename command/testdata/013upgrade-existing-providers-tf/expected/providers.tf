# This is a file called providers.tf which does not originally have a
# required_providers block. 
resource foo_resource a {}
terraform {
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
