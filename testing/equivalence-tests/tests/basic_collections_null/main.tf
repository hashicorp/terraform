terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

resource "tfcoremock_list" "list" {
  id = "985820B3-ACF9-4F00-94AD-F81C5EA33663"
}

resource "tfcoremock_set" "set" {
  id = "046952C9-B832-4106-82C0-C217F7C73E18"
}

resource "tfcoremock_map" "map" {
  id = "50E1A46E-E64A-4C1F-881C-BA85A5440964"
}
