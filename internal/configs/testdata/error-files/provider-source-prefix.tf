terraform {
  required_providers {
    usererror = {
      source = "foo/terraform-provider-foo" # ERROR: Invalid provider type
    }
    badname = {
      source = "foo/terraform-foo" # ERROR: Invalid provider type
    }
  }
}
