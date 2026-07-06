terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  provider_meta "test" {
    # Test fixture requires the mocked provider to have 'module_name' as an attribute
    # in the provider_meta schema.
    module_name = "foobar"
  }
}


provider "test" {}
