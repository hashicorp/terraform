terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}


resource "tfcoremock_simple_resource" "second" {
  string = "Hello, world!"
}

moved {
  from = tfcoremock_simple_resource.first
  to = tfcoremock_simple_resource.second
}
