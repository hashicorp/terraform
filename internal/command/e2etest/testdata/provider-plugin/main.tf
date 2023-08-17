// the provider-plugin tests uses the -plugin-cache flag so mnptu pulls the
// test binaries instead of reaching out to the registry.
mnptu {
  required_providers {
    simple5 = {
      source = "registry.mnptu.io/hashicorp/simple"
    }
    simple6 = {
      source = "registry.mnptu.io/hashicorp/simple6"
    }
  }
}

resource "simple_resource" "test-proto5" {
  provider = simple5
}

resource "simple_resource" "test-proto6" {
  provider = simple6
}
