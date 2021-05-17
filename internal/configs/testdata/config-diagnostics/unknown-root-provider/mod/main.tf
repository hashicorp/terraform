terraform {
  required_providers {
    bar = {
      version = "~>1.0.0"
    }
  }
}

resource "bar_resource" "x" {
}
