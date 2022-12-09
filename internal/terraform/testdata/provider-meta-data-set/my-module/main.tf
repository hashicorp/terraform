terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

data "test_file" "foo" {
  id = "bar"
}

terraform {
  provider_meta "test" {
    baz = "quux-submodule"
  }
}
