terraform {
  required_providers {
    simple = {
      source = "hashicorp/test"
    }
  }
}

resource "simple_resource" "test" {
}
