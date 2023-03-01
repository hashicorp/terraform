terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_object" "object" {
  object = {
    string  = "Hello, a totally different world!"
    boolean = false
    number  = 2
  }
}
