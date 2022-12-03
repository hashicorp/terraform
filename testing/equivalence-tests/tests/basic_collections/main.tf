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
    "3EC6EB1F-E372-46C3-A069-00D6E82EC1E1",
    "D01290F6-2D3A-45FA-B006-DAA80F6D31F6",
  ]
}

resource "tfcoremock_set" "set" {
  id = "046952C9-B832-4106-82C0-C217F7C73E18"
  set = [
    "41471135-E14C-4946-BFA4-2626C7E2A94A",
    "C04762B9-D07B-40FE-A92B-B72AD342658D",
    "D8F7EA80-9E25-4DD7-8D97-797D2080952B",
  ]
}

resource "tfcoremock_map" "map" {
  id = "50E1A46E-E64A-4C1F-881C-BA85A5440964"
  map = {
    "zero" : "6B044AF7-172B-495B-BE11-B9546C12C3BD",
    "one" : "682672C7-0918-4448-8342-887BAE01062A",
    "two" : "212FFBF6-40FE-4862-B708-E6AA508E84E0",
  }
}
