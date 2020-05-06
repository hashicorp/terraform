resource bar_instance a {}
resource baz_instance b {}
terraform {
  required_version = "> 0.12.0"
  required_providers {
    bar = {
      source = "registry.acme.corp/acme/bar"
    }
  }
}
