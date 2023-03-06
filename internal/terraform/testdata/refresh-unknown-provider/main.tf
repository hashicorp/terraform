terraform {
  required_providers {
    unknown = {
      source = "hashicorp/unknown"
    }
  }
}

resource "unknown_instance" "foo" {
    num = "2"
    compute = "foo"
}
