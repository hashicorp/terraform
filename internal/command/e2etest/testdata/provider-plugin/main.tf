// the provider-plugin tests uses the -plugin-cache flag so terraform pulls the
// test binaries instead of reaching out to the registry.
terraform {
  required_providers {
    simple5 = {
      source = "registry.terraform.io/hashicorp/simple"
    }
    simple6 = {
      source = "registry.terraform.io/hashicorp/simple6"
    }
  }
}

resource "simple_resource" "test-proto5" {
  provider = simple5
}

resource "simple_resource" "test-proto6" {
  provider = simple6
}
