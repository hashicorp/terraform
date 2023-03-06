terraform {
  required_providers {
    my-provider = {
      source = "terraform.io/test-only/provider"
    }
  }
}

terraform {
  provider_meta "my-provider" {
    hello = "test-module"
  }
}
