// the provider-plugin tests uses the -plugin-cache flag so terraform pulls the
// test binaries instead of reaching out to the registry.
terraform {
  required_providers {
    simple = {
      source = "registry.terraform.io/hashicorp/simple"
    }
  }
}

resource "simple_resource" "test-proto6" {
  // The schema capability setting of this provider will not allow
  // SchemaConfigModeAttr fixups, so this should produce an error.
  list {
    foo = "bar"
  }
}
