terraform {
  required_providers {
    tfcoremock = {
      source = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_simple_resource" "example" {
  id      = "my-simple-resource"
}
