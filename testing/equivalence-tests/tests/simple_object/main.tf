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
  id = "AF9833AE-3434-4D0B-8B69-F4B992565D9F"
  object = {
    string  = "Hello, world!"
    boolean = true
    number  = 10
  }
}
