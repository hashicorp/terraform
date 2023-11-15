terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_simple_resource" "base" {
  string = "Hello, change!"
  number = 0
}

resource "tfcoremock_simple_resource" "dependent" {
  string = tfcoremock_simple_resource.base.string
}
