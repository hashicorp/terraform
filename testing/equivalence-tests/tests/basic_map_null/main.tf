terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_map" "map" {
  id = "50E1A46E-E64A-4C1F-881C-BA85A5440964"
}
