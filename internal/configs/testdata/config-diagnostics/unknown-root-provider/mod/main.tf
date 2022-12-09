terraform {
  required_providers {
    null = {
      version = "~>1.0.0"
    }
  }
}

resource "null_resource" "x" {
}
