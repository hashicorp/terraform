terraform {
  required_providers {
    usererror = {
      source = "foo/terraform-provider-foo"
    }
    badname = {
      source = "foo/terraform-foo"
    }
  }
}
