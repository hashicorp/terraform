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
