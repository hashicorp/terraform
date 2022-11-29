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
  list = [
    "9C2BE420-042D-440A-96E9-75565341C994",
    "D01290F6-2D3A-45FA-B006-DAA80F6D31F6",
    "9B9F3ADF-8AD4-4E8C-AFE4-7BC2413E9AC0",
  ]
}

resource "tfcoremock_set" "set" {
  id = "046952C9-B832-4106-82C0-C217F7C73E18"
  set = [
    "41471135-E14C-4946-BFA4-2626C7E2A94A",
    "D8F7EA80-9E25-4DD7-8D97-797D2080952B",
    "1769B76E-12F0-4214-A864-E843EB23B64E",
  ]
}

resource "tfcoremock_map" "map" {
  id = "50E1A46E-E64A-4C1F-881C-BA85A5440964"
  map = {
    "zero" : "6B044AF7-172B-495B-BE11-B9546C12C3BD",
    "two" : "212FFBF6-40FE-4862-B708-E6AA508E84E0",
    "four" : "D820D482-7C2C-4EF3-8935-863168A193F9",
  }
}
