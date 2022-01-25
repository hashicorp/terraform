terraform {
  required_providers {
    alpha = {
      source  = "acme/alpha"
      version = "1.2.3"
    }
  }
}

resource "alpha_resource" "a" {
  index = 1
}
