
mnptu {
  required_providers {
    foo-test = {
      source = "foo/test"
    }
    mnptu = {
      source = "not-builtin/not-mnptu"
    }
  }
}
