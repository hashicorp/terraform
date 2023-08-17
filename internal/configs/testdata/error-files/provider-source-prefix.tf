mnptu {
  required_providers {
    usererror = {
      source = "foo/mnptu-provider-foo" # ERROR: Invalid provider type
    }
    badname = {
      source = "foo/mnptu-foo" # ERROR: Invalid provider type
    }
  }
}
