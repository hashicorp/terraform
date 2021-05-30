
terraform {
  required_providers {
    foo-test = {
      source = "foo/test"
    }
    terraform = {
      source = "not-builtin/not-terraform"
    }
  }
}
