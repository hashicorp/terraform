terraform {
  required_providers {
    foo = "0.5"
  }
}
terraform {
  required_providers {
    bar = {
      source = "registry.acme.corp/acme/bar"
    }
  }
}
