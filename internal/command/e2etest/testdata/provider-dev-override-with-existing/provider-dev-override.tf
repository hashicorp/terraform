terraform {
  required_providers {
    # this one is overwritten by dev override
    simple = {
      source  = "example.com/test/test"
      version = "2.0.0"
    }

    # this one should still be loaded
    null = {
      # Our version is intentionally fixed so that we have a fixed
      # test case here, though we might have to update this in future
      # if e.g. Terraform stops supporting plugin protocol 5, or if
      # the null provider is yanked from the registry for some reason.
      source  = "hashicorp/null"
      version = "3.1.0"
    }
  }
}

data "simple_resource" "test" {
}
