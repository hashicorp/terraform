terraform {
  required_providers {
    usererror = { # ERROR: Invalid provider type
      source = "foo/terraform-provider-foo"
    }
    badname = { # ERROR: Invalid provider type
      source = "foo/terraform-foo"
    }
  }
}
