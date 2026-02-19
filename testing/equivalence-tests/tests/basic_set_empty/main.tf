terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_set" "set" {
  id = "046952C9-B832-4106-82C0-C217F7C73E18"
  set = []
}
